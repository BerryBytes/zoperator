package usecase

import (
	"context"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UseCase interface {
	ReconcileNamespace(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error
	ReconcileResourceQuota(ctx context.Context, userConfig *myoperatorv1alpha1.UserConfig) error

	ReconcileSealedSecrets(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error
	ReconcileRBAC(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error
	ReconcileRole(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error
	ReconcileServiceAccount(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error
	ReconcileRoleBinding(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error
	ReconcileLimitRange(ctx context.Context, userConfig *myoperatorv1alpha1.UserConfig) error
	GenerateAndSaveKubeconfig(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error
	ReconcileNetworkPolicies(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error

	HandleDeletion(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) (ctrl.Result, error)
}

type UserConfigUseCase struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewUserConfigUseCase(client client.Client, scheme *runtime.Scheme) UseCase {
	return &UserConfigUseCase{
		Client: client,
		Scheme: scheme,
	}
}
