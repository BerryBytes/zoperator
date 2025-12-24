package usecase

import (
	"context"
	"fmt"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"
)

func (u *UserConfigUseCase) ReconcileNamespace(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: uc.Name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":          "userconfig-operator",
				"userconfig.myoperator.01cloud.io/name": uc.Name,
			},
		},
	}

	// Set ownership reference
	if err := controllerutil.SetControllerReference(uc, namespace, u.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	// Create namespace if it doesn't exist
	if err := u.Create(ctx, namespace); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Attach the ResourceQuota
	if err := u.ReconcileResourceQuota(ctx, uc); err != nil {
		return fmt.Errorf("failed to reconcile ResourceQuota for namespace %s: %w", uc.Name, err)
	}

	return nil
}
