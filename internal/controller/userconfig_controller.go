package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	sealedsecretsv1alpha1 "github.com/bitnami-labs/sealed-secrets/pkg/apis/sealedsecrets/v1alpha1"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"
	usecase "01cloud/zoperator/internal/usecase"
)

// UserConfigReconciler reconciles a UserConfig object
type UserConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	UC     usecase.UseCase
}

const (
	// userConfigFinalizer is the finalizer name used to prevent premature deletion
	userConfigFinalizer = "myoperator.01cloud.io/finalizer"

	// Error messages
	errGetUserConfig    = "failed to get UserConfig"
	errUpdateFinalizer  = "failed to update finalizer"
	errReconcileNS      = "failed to reconcile namespace"
	errGetNamespace     = "failed to get namespace"
	errReconcileSecrets = "failed to reconcile sealed secrets"
	errUpdateStatus     = "failed to update status"
)

// RBAC permissions required for the controller
// +kubebuilder:rbac:groups=myoperator.01cloud.io,resources=userconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=myoperator.01cloud.io,resources=userconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=myoperator.01cloud.io,resources=userconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=bitnami.com,resources=sealedsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=resourcequotas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=resourcequotas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=limitranges,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts/token,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=bitnami.com,resources=sealedsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;delete;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=*
// +kubebuilder:rbac:groups="",resources=configmap,verbs=*

// +kubebuilder:rbac:groups=core,resources=pods/log,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/scale,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets/scale,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets/scale,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/scale,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaim,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaim,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=persistentvolume,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=core,resources=persistentvolumes/status,verbs=get;update;patch

// Reconcile handles the reconciliation loop for UserConfig resources
func (r *UserConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get UserConfig instance
	userConfig := &myoperatorv1alpha1.UserConfig{}
	if err := r.Get(ctx, req.NamespacedName, userConfig); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !userConfig.DeletionTimestamp.IsZero() {
		// If the finalizer is not present, return
		if !controllerutil.ContainsFinalizer(userConfig, userConfigFinalizer) {
			return ctrl.Result{}, nil
		}
		_, err := r.UC.HandleDeletion(ctx, userConfig)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(userConfig, userConfigFinalizer) {
		controllerutil.AddFinalizer(userConfig, userConfigFinalizer)
		if err := r.Update(ctx, userConfig); err != nil {
			userConfig = r.updateErrorStatus(ctx, userConfig, fmt.Errorf("Failed to update finalizer: %v", err))
			return ctrl.Result{}, err
		}
	}

	// Create/Update namespace
	if err := r.UC.ReconcileNamespace(ctx, userConfig); err != nil {
		r.updateErrorStatus(ctx, userConfig, fmt.Errorf("Failed to reconcile namespace: %v", err))
		return ctrl.Result{}, err
	}

	// Create/Update sealed secrets
	if err := r.UC.ReconcileSealedSecrets(ctx, userConfig); err != nil {
		userConfig = r.updateErrorStatus(ctx, userConfig, fmt.Errorf("Failed to reconcile sealed secrets: %v", err))
		return ctrl.Result{}, err
	}

	// After namespace reconciliation
	if err := r.UC.ReconcileRBAC(ctx, userConfig); err != nil {
		userConfig = r.updateErrorStatus(ctx, userConfig, fmt.Errorf("Failed to reconcile RBAC: %v", err))
		return ctrl.Result{}, err
	}

	if err := r.UC.ReconcileLimitRange(ctx, userConfig); err != nil {
		userConfig = r.updateErrorStatus(ctx, userConfig, fmt.Errorf("Failed to reconcile LimitRange: %v", err))
		return ctrl.Result{}, err
	}

	// Generate and save kubeconfig
	if err := r.UC.GenerateAndSaveKubeconfig(ctx, userConfig); err != nil {
		userConfig = r.updateErrorStatus(ctx, userConfig, fmt.Errorf("Failed to generate and save kubeconfig: %v", err))
		return ctrl.Result{}, err
	}

	// create network policy
	if err := r.UC.ReconcileNetworkPolicies(ctx, userConfig); err != nil {
		userConfig = r.updateErrorStatus(ctx, userConfig, fmt.Errorf("Failed to reconcile network policies: %v", err))
		return ctrl.Result{}, err
	}

	userConfig.Status.State = "Active"
	userConfig.Status.Conditions = append(userConfig.Status.Conditions, metav1.Condition{
		Type:               myoperatorv1alpha1.ReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciled",
		Message:            "UserConfig reconciled successfully",
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, userConfig); err != nil {
		log.FromContext(ctx).Error(err, errUpdateStatus)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// updateErrorStatus updates UserConfig status to Error state
func (r *UserConfigReconciler) updateErrorStatus(ctx context.Context, userConfig *myoperatorv1alpha1.UserConfig, err error) *myoperatorv1alpha1.UserConfig {
	userConfig.Status.State = "Error"
	userConfig.Status.Conditions = append(userConfig.Status.Conditions, metav1.Condition{
		Type:               myoperatorv1alpha1.ReadyCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "Error",
		Message:            err.Error(),
		LastTransitionTime: metav1.Now(),
	})
	if statusErr := r.Status().Update(ctx, userConfig); statusErr != nil {
		log.FromContext(ctx).Error(statusErr, "Failed to update status")
		return userConfig
	}
	return userConfig
}

// SetupWithManager sets up the controller with the Manager
func (r *UserConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&myoperatorv1alpha1.UserConfig{}).
		Owns(&corev1.Namespace{}).
		Owns(&sealedsecretsv1alpha1.SealedSecret{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
