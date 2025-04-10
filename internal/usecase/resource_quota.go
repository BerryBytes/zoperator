package usecase

import (
	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"
	"context"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (u *UserConfigUseCase) ReconcileResourceQuota(ctx context.Context, userConfig *myoperatorv1alpha1.UserConfig) error {
	resourceQuota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      userConfig.Name,
			Namespace: userConfig.Name,
		},
	}

	// Check if a custom ResourceQuota is specified in the UserConfig
	if userConfig.Spec.ResourceQuotas != nil {
		// Convert the custom ResourceQuota struct to ResourceQuota spec
		hardLimits := make(corev1.ResourceList)

		rq := userConfig.Spec.ResourceQuotas

		// Add each field from the struct to the hardLimits map if it's not empty
		if rq.CPU != "" {
			hardLimits[corev1.ResourceName("cpu")] = resource.MustParse(rq.CPU)
		}
		if rq.Memory != "" {
			hardLimits[corev1.ResourceName("memory")] = resource.MustParse(rq.Memory)
		}
		if rq.EphemeralStorage != "" {
			hardLimits[corev1.ResourceName("ephemeral-storage")] = resource.MustParse(rq.EphemeralStorage)
		}
		if rq.RequestsCPU != "" {
			hardLimits[corev1.ResourceName("requests.cpu")] = resource.MustParse(rq.RequestsCPU)
		}
		if rq.RequestsMemory != "" {
			hardLimits[corev1.ResourceName("requests.memory")] = resource.MustParse(rq.RequestsMemory)
		}
		if rq.RequestsStorage != "" {
			hardLimits[corev1.ResourceName("requests.storage")] = resource.MustParse(rq.RequestsStorage)
		}
		if rq.RequestsEphemeralStorage != "" {
			hardLimits[corev1.ResourceName("requests.ephemeral-storage")] = resource.MustParse(rq.RequestsEphemeralStorage)
		}
		if rq.LimitsCPU != "" {
			hardLimits[corev1.ResourceName("limits.cpu")] = resource.MustParse(rq.LimitsCPU)
		}
		if rq.LimitsMemory != "" {
			hardLimits[corev1.ResourceName("limits.memory")] = resource.MustParse(rq.LimitsMemory)
		}
		if rq.LimitsEphemeralStorage != "" {
			hardLimits[corev1.ResourceName("limits.ephemeral-storage")] = resource.MustParse(rq.LimitsEphemeralStorage)
		}
		if rq.Pods != "" {
			hardLimits[corev1.ResourceName("pods")] = resource.MustParse(rq.Pods)
		}
		if rq.Services != "" {
			hardLimits[corev1.ResourceName("services")] = resource.MustParse(rq.Services)
		}
		if rq.ReplicationControllers != "" {
			hardLimits[corev1.ResourceName("replicationcontrollers")] = resource.MustParse(rq.ReplicationControllers)
		}
		if rq.Secrets != "" {
			hardLimits[corev1.ResourceName("secrets")] = resource.MustParse(rq.Secrets)
		}
		if rq.ConfigMaps != "" {
			hardLimits[corev1.ResourceName("configmaps")] = resource.MustParse(rq.ConfigMaps)
		}
		if rq.PersistentVolumeClaims != "" {
			hardLimits[corev1.ResourceName("persistentvolumeclaims")] = resource.MustParse(rq.PersistentVolumeClaims)
		}
		if rq.ServicesNodePorts != "" {
			hardLimits[corev1.ResourceName("services.nodeports")] = resource.MustParse(rq.ServicesNodePorts)
		}
		if rq.ServicesLoadBalancers != "" {
			hardLimits[corev1.ResourceName("services.loadbalancers")] = resource.MustParse(rq.ServicesLoadBalancers)
		}

		// Set the resource quota specification
		resourceQuota.Spec = corev1.ResourceQuotaSpec{
			Hard: hardLimits,
		}
	} else {
		// Attach the default ResourceQuota if none is specified in the UserConfig
		resourceQuota.Spec = corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourcePods:   resource.MustParse("10"),
				corev1.ResourceCPU:    resource.MustParse("2"),
				corev1.ResourceMemory: resource.MustParse("4Gi"),
			},
		}
	}

	existingQuota := &corev1.ResourceQuota{}
	err := u.Client.Get(ctx, client.ObjectKey{Name: userConfig.Name, Namespace: userConfig.Name}, existingQuota)
	if err != nil && apierrors.IsNotFound(err) {
		// ResourceQuota doesn't exist, create it
		if err := u.Client.Create(ctx, resourceQuota); err != nil {
			return fmt.Errorf("failed to create ResourceQuota in namespace %s: %w", userConfig.Name, err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get ResourceQuota in namespace %s: %w", userConfig.Name, err)
	} else {
		// ResourceQuota exists, check if it needs to be updated
		if !reflect.DeepEqual(existingQuota.Spec, resourceQuota.Spec) {
			existingQuota.Spec = resourceQuota.Spec
			if err := u.Client.Update(ctx, existingQuota); err != nil {
				return fmt.Errorf("failed to update ResourceQuota in namespace %s: %w", userConfig.Name, err)
			}
		}
	}
	return nil
}
