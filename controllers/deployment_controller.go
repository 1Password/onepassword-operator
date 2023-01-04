/*
MIT License

Copyright (c) 2020-2022 1Password

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controllers

import (
	"context"
	"fmt"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/1Password/connect-sdk-go/connect"

	kubeSecrets "github.com/1Password/onepassword-operator/pkg/kubernetessecrets"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"
	"github.com/1Password/onepassword-operator/pkg/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logDeployment = logf.Log.WithName("controller_deployment")

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	OpConnectClient    connect.Client
	OpAnnotationRegExp *regexp.Regexp
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OnePasswordItem object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := logDeployment.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Deployment")

	deployment := &appsv1.Deployment{}
	err := r.Get(context.Background(), req.NamespacedName, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	annotations, annotationsFound := op.GetAnnotationsForDeployment(deployment, r.OpAnnotationRegExp)
	if !annotationsFound {
		reqLogger.Info("No 1Password Annotations found")
		return ctrl.Result{}, nil
	}

	//If the deployment is not being deleted
	if deployment.ObjectMeta.DeletionTimestamp.IsZero() {
		// Adds a finalizer to the deployment if one does not exist.
		// This is so we can handle cleanup of associated secrets properly
		if !utils.ContainsString(deployment.ObjectMeta.Finalizers, finalizer) {
			deployment.ObjectMeta.Finalizers = append(deployment.ObjectMeta.Finalizers, finalizer)
			if err := r.Update(context.Background(), deployment); err != nil {
				return reconcile.Result{}, err
			}
		}
		// Handles creation or updating secrets for deployment if needed
		if err := r.handleApplyingDeployment(deployment, deployment.Namespace, annotations, req); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// The deployment has been marked for deletion. If the one password
	// finalizer is found there are cleanup tasks to perform
	if utils.ContainsString(deployment.ObjectMeta.Finalizers, finalizer) {

		secretName := annotations[op.NameAnnotation]
		if err = r.cleanupKubernetesSecretForDeployment(secretName, deployment); err != nil {
			return ctrl.Result{}, err
		}

		// Remove the finalizer from the deployment so deletion of deployment can be completed
		if err := r.removeOnePasswordFinalizerFromDeployment(deployment); err != nil {
			return reconcile.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(r)
}

func (r *DeploymentReconciler) cleanupKubernetesSecretForDeployment(secretName string, deletedDeployment *appsv1.Deployment) error {
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
		if err := r.Delete(context.Background(), kubernetesSecret); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

func (r *DeploymentReconciler) areMultipleDeploymentsUsingSecret(updatedSecrets map[string]*corev1.Secret, deletedDeployment appsv1.Deployment) (bool, error) {
	deployments := &appsv1.DeploymentList{}
	opts := []client.ListOption{
		client.InNamespace(deletedDeployment.Namespace),
	}

	err := r.List(context.Background(), deployments, opts...)
	if err != nil {
		logDeployment.Error(err, "Failed to list kubernetes deployments")
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

func (r *DeploymentReconciler) removeOnePasswordFinalizerFromDeployment(deployment *appsv1.Deployment) error {
	deployment.ObjectMeta.Finalizers = utils.RemoveString(deployment.ObjectMeta.Finalizers, finalizer)
	return r.Update(context.Background(), deployment)
}

func (r *DeploymentReconciler) handleApplyingDeployment(deployment *appsv1.Deployment, namespace string, annotations map[string]string, request reconcile.Request) error {
	reqLog := logDeployment.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)

	secretName := annotations[op.NameAnnotation]
	secretLabels := map[string]string(nil)
	secretType := string(corev1.SecretTypeOpaque)

	if len(secretName) == 0 {
		reqLog.Info("No 'item-name' annotation set. 'item-path' and 'item-name' must be set as annotations to add new secret.")
		return nil
	}

	item, err := op.GetOnePasswordItemByPath(r.OpConnectClient, annotations[op.ItemPathAnnotation])
	if err != nil {
		return fmt.Errorf("Failed to retrieve item: %v", err)
	}

	// Create owner reference.
	gvk, err := apiutil.GVKForObject(deployment, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not to retrieve group version kind: %v", err)
	}
	ownerRef := &metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       deployment.GetName(),
		UID:        deployment.GetUID(),
	}

	return kubeSecrets.CreateKubernetesSecretFromItem(r.Client, secretName, namespace, item, annotations[op.RestartDeploymentsAnnotation], secretLabels, secretType, ownerRef)
}
