/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/1Password/onepassword-operator/pkg/onepassword"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"

	kubeSecrets "github.com/1Password/onepassword-operator/pkg/kubernetessecrets"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/1Password/onepassword-operator/pkg/utils"

	"github.com/1Password/connect-sdk-go/connect"
	corev1 "k8s.io/api/core/v1"

	onepasswordv1 "github.com/1Password/onepassword-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var log = logf.Log.WithName("controller_onepassworditem")

// OnePasswordItemReconciler reconciles a OnePasswordItem object
type OnePasswordItemReconciler struct {
	kubeClient      kubeClient.Client
	scheme          *runtime.Scheme
	opConnectClient connect.Client
}

//+kubebuilder:rbac:groups=onepassword.onepassword.com,resources=onepassworditems,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=onepassword.onepassword.com,resources=onepassworditems/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=onepassword.onepassword.com,resources=onepassworditems/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OnePasswordItem object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *OnePasswordItemReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling OnePasswordItem")

	onepassworditem := &onepasswordv1.OnePasswordItem{}
	err := r.kubeClient.Get(context.Background(), request.NamespacedName, onepassworditem)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// If the deployment is not being deleted
	if onepassworditem.ObjectMeta.DeletionTimestamp.IsZero() {
		// Adds a finalizer to the deployment if one does not exist.
		// This is so we can handle cleanup of associated secrets properly
		if !utils.ContainsString(onepassworditem.ObjectMeta.Finalizers, finalizer) {
			onepassworditem.ObjectMeta.Finalizers = append(onepassworditem.ObjectMeta.Finalizers, finalizer)
			if err := r.kubeClient.Update(context.Background(), onepassworditem); err != nil {
				return reconcile.Result{}, err
			}
		}

		// Handles creation or updating secrets for deployment if needed
		if err := r.HandleOnePasswordItem(onepassworditem, request); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}
	// If one password finalizer exists then we must cleanup associated secrets
	if utils.ContainsString(onepassworditem.ObjectMeta.Finalizers, finalizer) {

		// Delete associated kubernetes secret
		if err = r.cleanupKubernetesSecret(onepassworditem); err != nil {
			return reconcile.Result{}, err
		}

		// Remove finalizer now that cleanup is complete
		if err := r.removeFinalizer(onepassworditem); err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OnePasswordItemReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New("onepassworditem-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource OnePasswordItem
	err = c.Watch(&source.Kind{Type: &onepasswordv1.OnePasswordItem{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
	// TODO Consider the simplified code below. Based on the migration guide: https://sdk.operatorframework.io/docs/building-operators/golang/migration/#create-a-new-project
	//	return ctrl.NewControllerManagedBy(mgr).Named("onepassworditem-controller").WithOptions(controller.Options{Reconciler: r}).
	//		For(&onepasswordv1.OnePasswordItem{}).Watches(&source.Kind{Type: &onepasswordv1.OnePasswordItem{}}, &handler.EnqueueRequestForObject{}).
	//		Complete(r)
}

func (r *OnePasswordItemReconciler) removeFinalizer(onePasswordItem *onepasswordv1.OnePasswordItem) error {
	onePasswordItem.ObjectMeta.Finalizers = utils.RemoveString(onePasswordItem.ObjectMeta.Finalizers, finalizer)
	if err := r.kubeClient.Update(context.Background(), onePasswordItem); err != nil {
		return err
	}
	return nil
}

func (r *OnePasswordItemReconciler) cleanupKubernetesSecret(onePasswordItem *onepasswordv1.OnePasswordItem) error {
	kubernetesSecret := &corev1.Secret{}
	kubernetesSecret.ObjectMeta.Name = onePasswordItem.Name
	kubernetesSecret.ObjectMeta.Namespace = onePasswordItem.Namespace

	r.kubeClient.Delete(context.Background(), kubernetesSecret)
	if err := r.kubeClient.Delete(context.Background(), kubernetesSecret); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (r *OnePasswordItemReconciler) HandleOnePasswordItem(resource *onepasswordv1.OnePasswordItem, request reconcile.Request) error {
	secretName := resource.GetName()
	labels := resource.Labels
	annotations := resource.Annotations
	secretType := resource.Type
	autoRestart := annotations[op.RestartDeploymentsAnnotation]

	item, err := onepassword.GetOnePasswordItemByPath(r.opConnectClient, resource.Spec.ItemPath)
	if err != nil {
		return fmt.Errorf("Failed to retrieve item: %v", err)
	}

	// Create owner reference.
	gvk, err := apiutil.GVKForObject(resource, r.scheme)
	if err != nil {
		return fmt.Errorf("could not to retrieve group version kind: %v", err)
	}
	ownerRef := &metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       resource.GetName(),
		UID:        resource.GetUID(),
	}

	return kubeSecrets.CreateKubernetesSecretFromItem(r.kubeClient, secretName, resource.Namespace, item, autoRestart, labels, secretType, annotations, ownerRef)
}
