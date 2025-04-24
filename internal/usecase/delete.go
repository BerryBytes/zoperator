package usecase

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"
)

func (u *UserConfigUseCase) HandleDeletion(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(uc, "myoperator.01cloud.io/finalizer") {
		return ctrl.Result{}, nil
	}

	// Delete namespace (this will cascade delete secrets)
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: uc.Name,
		},
	}
	if err := u.Delete(ctx, namespace); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(uc, "myoperator.01cloud.io/finalizer")
	if err := u.Update(ctx, uc); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
