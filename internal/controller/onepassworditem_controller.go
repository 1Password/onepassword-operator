/*
MIT License

Copyright (c) 2020-2024 1Password

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

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	onepasswordv1 "github.com/1Password/onepassword-operator/api/v1"
	kubeSecrets "github.com/1Password/onepassword-operator/pkg/kubernetessecrets"
	"github.com/1Password/onepassword-operator/pkg/logs"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"
	opclient "github.com/1Password/onepassword-operator/pkg/onepassword/client"
	"github.com/1Password/onepassword-operator/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logOnePasswordItem = logf.Log.WithName("controller_onepassworditem")
var finalizer = "onepassword.com/finalizer.secret"

// OnePasswordItemReconciler reconciles a OnePasswordItem object
type OnePasswordItemReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	OpClient opclient.Client
}

// +kubebuilder:rbac:groups=onepassword.com,resources=onepassworditems,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=onepassword.com,resources=onepassworditems/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=onepassword.com,resources=onepassworditems/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=pods,verbs=get
// +kubebuilder:rbac:groups="",resources=pods;services;services/finalizers;endpoints;persistentvolumeclaims;events;configmaps;secrets;namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets;deployments;replicasets;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets;deployments,verbs=get
// +kubebuilder:rbac:groups=apps,resourceNames=onepassword-connect-operator,resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=onepassword.com,resources=*,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;create
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OnePasswordItem object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile
func (r *OnePasswordItemReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := logOnePasswordItem.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.V(logs.DebugLevel).Info("Reconciling OnePasswordItem")

	onepassworditem := &onepasswordv1.OnePasswordItem{}
	err := r.Get(ctx, req.NamespacedName, onepassworditem)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// If the deployment is not being deleted
	if onepassworditem.ObjectMeta.DeletionTimestamp.IsZero() {
		// Adds a finalizer to the deployment if one does not exist.
		// This is so we can handle cleanup of associated secrets properly
		if !utils.ContainsString(onepassworditem.ObjectMeta.Finalizers, finalizer) {
			onepassworditem.ObjectMeta.Finalizers = append(onepassworditem.ObjectMeta.Finalizers, finalizer)
			if err = r.Update(ctx, onepassworditem); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Handles creation or updating secrets for deployment if needed
		err = r.handleOnePasswordItem(ctx, onepassworditem, req)
		if err != nil {
			if strings.Contains(err.Error(), "rate limit") {
				reqLogger.V(logs.InfoLevel).Info("1Password rate limit hit. Requeuing after 15 minutes.")
				return ctrl.Result{RequeueAfter: 15 * time.Minute}, nil
			}
		}
		if updateStatusErr := r.updateStatus(ctx, onepassworditem, err); updateStatusErr != nil {
			return ctrl.Result{}, fmt.Errorf("cannot update status: %s", updateStatusErr)
		}
		return ctrl.Result{}, err
	}
	// If one password finalizer exists then we must cleanup associated secrets
	if utils.ContainsString(onepassworditem.ObjectMeta.Finalizers, finalizer) {

		// Delete associated kubernetes secret
		if err = r.cleanupKubernetesSecret(ctx, onepassworditem); err != nil {
			return ctrl.Result{}, err
		}

		// Remove finalizer now that cleanup is complete
		if err = r.removeOnePasswordFinalizerFromOnePasswordItem(ctx, onepassworditem); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OnePasswordItemReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&onepasswordv1.OnePasswordItem{}).
		Named("onepassworditem").
		Complete(r)
}

func (r *OnePasswordItemReconciler) cleanupKubernetesSecret(ctx context.Context, onePasswordItem *onepasswordv1.OnePasswordItem) error {
	kubernetesSecret := &corev1.Secret{}
	kubernetesSecret.ObjectMeta.Name = onePasswordItem.Name
	kubernetesSecret.ObjectMeta.Namespace = onePasswordItem.Namespace

	if err := r.Delete(ctx, kubernetesSecret); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (r *OnePasswordItemReconciler) removeOnePasswordFinalizerFromOnePasswordItem(ctx context.Context, onePasswordItem *onepasswordv1.OnePasswordItem) error {
	onePasswordItem.ObjectMeta.Finalizers = utils.RemoveString(onePasswordItem.ObjectMeta.Finalizers, finalizer)
	return r.Update(ctx, onePasswordItem)
}

func (r *OnePasswordItemReconciler) handleOnePasswordItem(ctx context.Context, resource *onepasswordv1.OnePasswordItem, _ ctrl.Request) error {
	secretName := resource.GetName()
	labels := resource.Labels
	secretType := resource.Type
	autoRestart := resource.Annotations[op.RestartDeploymentsAnnotation]

	item, err := op.GetOnePasswordItemByPath(ctx, r.OpClient, resource.Spec.ItemPath)
	if err != nil {
		return fmt.Errorf("Failed to retrieve item: %v", err)
	}

	// Create owner reference.
	gvk, err := apiutil.GVKForObject(resource, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not to retrieve group version kind: %v", err)
	}
	ownerRef := &metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       resource.GetName(),
		UID:        resource.GetUID(),
	}

	return kubeSecrets.CreateKubernetesSecretFromItem(ctx, r.Client, secretName, resource.Namespace, item, autoRestart, labels, secretType, ownerRef)
}

func (r *OnePasswordItemReconciler) updateStatus(ctx context.Context, resource *onepasswordv1.OnePasswordItem, err error) error {
	existingCondition := findCondition(resource.Status.Conditions, onepasswordv1.OnePasswordItemReady)
	updatedCondition := existingCondition
	if err != nil {
		updatedCondition.Message = err.Error()
		updatedCondition.Status = metav1.ConditionFalse
	} else {
		updatedCondition.Message = ""
		updatedCondition.Status = metav1.ConditionTrue
	}

	if existingCondition.Status != updatedCondition.Status {
		updatedCondition.LastTransitionTime = metav1.Now()
	}

	resource.Status.Conditions = []onepasswordv1.OnePasswordItemCondition{updatedCondition}
	return r.Status().Update(ctx, resource)
}

func findCondition(conditions []onepasswordv1.OnePasswordItemCondition, t onepasswordv1.OnePasswordItemConditionType) onepasswordv1.OnePasswordItemCondition {
	for _, c := range conditions {
		if c.Type == t {
			return c
		}
	}
	return onepasswordv1.OnePasswordItemCondition{
		Type:   t,
		Status: metav1.ConditionUnknown,
	}
}
