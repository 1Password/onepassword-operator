package deployment

import (
	"context"
	"fmt"

	kubeSecrets "github.com/1Password/onepassword-operator/pkg/kubernetessecrets"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"
	"github.com/1Password/onepassword-operator/pkg/utils"

	"regexp"

	"github.com/1Password/connect-sdk-go/connect"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_deployment")
var finalizer = "onepassword.com/finalizer.secret"

const annotationRegExpString = "^operator.1password.io\\/[a-zA-Z\\.]+"

func Add(mgr manager.Manager, opConnectClient connect.Client) error {
	return add(mgr, newReconciler(mgr, opConnectClient))
}

func newReconciler(mgr manager.Manager, opConnectClient connect.Client) *ReconcileDeployment {
	r, _ := regexp.Compile(annotationRegExpString)
	return &ReconcileDeployment{
		opAnnotationRegExp: r,
		kubeClient:         mgr.GetClient(),
		scheme:             mgr.GetScheme(),
		opConnectClient:    opConnectClient,
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New("deployment-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Deployment
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileDeployment{}

type ReconcileDeployment struct {
	opAnnotationRegExp *regexp.Regexp
	kubeClient         client.Client
	scheme             *runtime.Scheme
	opConnectClient    connect.Client
}

func (r *ReconcileDeployment) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(r)
}

func (r *ReconcileDeployment) test() {
	return
}

// Reconcile reads that state of the cluster for a Deployment object and makes changes based on the state read
// and what is in the Deployment.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Deployment")

	deployment := &appsv1.Deployment{}
	err := r.kubeClient.Get(ctx, request.NamespacedName, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	annotations, annotationsFound := op.GetAnnotationsForDeployment(deployment, r.opAnnotationRegExp)
	if !annotationsFound {
		reqLogger.Info("No 1Password Annotations found")
		return reconcile.Result{}, nil
	}

	//If the deployment is not being deleted
	if deployment.ObjectMeta.DeletionTimestamp.IsZero() {
		// Adds a finalizer to the deployment if one does not exist.
		// This is so we can handle cleanup of associated secrets properly
		if !utils.ContainsString(deployment.ObjectMeta.Finalizers, finalizer) {
			deployment.ObjectMeta.Finalizers = append(deployment.ObjectMeta.Finalizers, finalizer)
			if err := r.kubeClient.Update(context.Background(), deployment); err != nil {
				return reconcile.Result{}, err
			}
		}
		// Handles creation or updating secrets for deployment if needed
		if err := r.HandleApplyingDeployment(deployment, deployment.Namespace, annotations, request); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}
	// The deployment has been marked for deletion. If the one password
	// finalizer is found there are cleanup tasks to perform
	if utils.ContainsString(deployment.ObjectMeta.Finalizers, finalizer) {

		secretName := annotations[op.NameAnnotation]
		r.cleanupKubernetesSecretForDeployment(secretName, deployment)

		// Remove the finalizer from the deployment so deletion of deployment can be completed
		if err := r.removeOnePasswordFinalizerFromDeployment(deployment); err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileDeployment) cleanupKubernetesSecretForDeployment(secretName string, deletedDeployment *appsv1.Deployment) error {
	kubernetesSecret := &corev1.Secret{}
	kubernetesSecret.ObjectMeta.Name = secretName
	kubernetesSecret.ObjectMeta.Namespace = deletedDeployment.Namespace

	if len(secretName) == 0 {
		return nil
	}
	updatedSecrets := map[string]*corev1.Secret{secretName: kubernetesSecret}

	multipleDeploymentsUsingSecret, err := r.areMultipleDeploymentsUsingSecret(updatedSecrets, *deletedDeployment)
	if err != nil {
		return err
	}

	// Only delete the associated kubernetes secret if it is not being used by other deployments
	if !multipleDeploymentsUsingSecret {
		if err := r.kubeClient.Delete(context.Background(), kubernetesSecret); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func (r *ReconcileDeployment) areMultipleDeploymentsUsingSecret(updatedSecrets map[string]*corev1.Secret, deletedDeployment appsv1.Deployment) (bool, error) {
	deployments := &appsv1.DeploymentList{}
	opts := []client.ListOption{
		client.InNamespace(deletedDeployment.Namespace),
	}

	err := r.kubeClient.List(context.Background(), deployments, opts...)
	if err != nil {
		log.Error(err, "Failed to list kubernetes deployments")
		return false, err
	}

	for i := 0; i < len(deployments.Items); i++ {
		if deployments.Items[i].Name != deletedDeployment.Name {
			if op.IsDeploymentUsingSecrets(&deployments.Items[i], updatedSecrets) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (r *ReconcileDeployment) removeOnePasswordFinalizerFromDeployment(deployment *appsv1.Deployment) error {
	deployment.ObjectMeta.Finalizers = utils.RemoveString(deployment.ObjectMeta.Finalizers, finalizer)
	return r.kubeClient.Update(context.Background(), deployment)
}

func (r *ReconcileDeployment) HandleApplyingDeployment(deployment *appsv1.Deployment, namespace string, annotations map[string]string, request reconcile.Request) error {
	reqLog := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	secretName := annotations[op.NameAnnotation]
	secretLabels := map[string]string(nil)
	secretType := ""

	if len(secretName) == 0 {
		reqLog.Info("No 'item-name' annotation set. 'item-path' and 'item-name' must be set as annotations to add new secret.")
		return nil
	}

	item, err := op.GetOnePasswordItemByPath(r.opConnectClient, annotations[op.ItemPathAnnotation])
	if err != nil {
		return fmt.Errorf("Failed to retrieve item: %v", err)
	}

	// Create owner reference.
	gvk, err := apiutil.GVKForObject(deployment, r.scheme)
	if err != nil {
		return fmt.Errorf("could not to retrieve group version kind: %v", err)
	}
	ownerRef := &metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       deployment.GetName(),
		UID:        deployment.GetUID(),
	}

	return kubeSecrets.CreateKubernetesSecretFromItem(r.kubeClient, secretName, namespace, item, annotations[op.RestartDeploymentsAnnotation], secretLabels, secretType, ownerRef)
}
