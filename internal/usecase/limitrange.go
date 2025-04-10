package usecase

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"
)

func (u *UserConfigUseCase) ReconcileLimitRange(ctx context.Context, userConfig *myoperatorv1alpha1.UserConfig) error {
	defaultLimits := corev1.LimitRangeItem{
		Type: corev1.LimitTypeContainer,
		Default: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
		DefaultRequest: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Min: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("50m"),
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		},
		Max: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("4Gi"),
		},
	}

	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userConfig.Name,
			Namespace: userConfig.Name,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{},
		},
	}
	if userConfig.Spec.LimitRange == nil || len(userConfig.Spec.LimitRange.Limits) == 0 {
		limitRange.Spec.Limits = []corev1.LimitRangeItem{defaultLimits}
	} else {
		for _, userLimits := range userConfig.Spec.LimitRange.Limits {
			item := corev1.LimitRangeItem{
				Type: corev1.LimitType(userLimits.Type),
				Max:  parseResourceList(userLimits.Max, defaultLimits.Max),
				Min:  parseResourceList(userLimits.Min, defaultLimits.Min),
			}
			if userLimits.Type != string(corev1.LimitTypePod) {
				item.Default = parseResourceList(userLimits.Default, defaultLimits.Default)
				item.DefaultRequest = parseResourceList(userLimits.DefaultRequest, defaultLimits.DefaultRequest)
			}
			limitRange.Spec.Limits = append(limitRange.Spec.Limits, item)
		}
	}

	// Set controller reference
	if err := controllerutil.SetControllerReference(userConfig, limitRange, u.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for LimitRange: %w", err)
	}

	// Create or update the LimitRange
	existing := &corev1.LimitRange{}
	err := u.Get(ctx, client.ObjectKey{Namespace: userConfig.Name, Name: limitRange.Name}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		if err := u.Create(ctx, limitRange); err != nil {
			return fmt.Errorf("failed to create LimitRange: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get LimitRange: %w", err)
	} else {
		existing.Spec = limitRange.Spec
		if err := u.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update LimitRange: %w", err)
		}
	}

	return nil
}

func parseResourceList(input *myoperatorv1alpha1.Resources, fallback corev1.ResourceList) corev1.ResourceList {
	if input == nil {
		return fallback
	}
	return corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse(input.CPU),
		corev1.ResourceMemory: resource.MustParse(input.Memory),
	}
}
