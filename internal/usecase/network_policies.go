package usecase

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"
)

func (u *UserConfigUseCase) ReconcileNetworkPolicies(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error {
	netpol := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uc.Name,
			Namespace: uc.Name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":          "userconfig-operator",
				"userconfig.myoperator.01cloud.io/name": uc.Name,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{}, // Applies to all pods in namespace
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
		},
	}

	if len(uc.Spec.NetworkPolicy) == 0 {
		// Create a default deny policy
		// Empty ingress and egress arrays = deny all
		netpol.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{}
		netpol.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{}
	} else {
		// Combine all allowed traffic rules into a single policy
		var ingressRules []networkingv1.NetworkPolicyIngressRule
		var egressRules []networkingv1.NetworkPolicyEgressRule

		for _, policy := range uc.Spec.NetworkPolicy {
			// Configure ingress rules if allowTrafficFrom is specified
			if policy.AllowTrafficFrom != nil {
				ingressRule := networkingv1.NetworkPolicyIngressRule{}

				// Configure pod selector
				if len(policy.AllowTrafficFrom.Pods) > 0 {
					for _, podSelector := range policy.AllowTrafficFrom.Pods {
						ingressRule.From = append(ingressRule.From, networkingv1.NetworkPolicyPeer{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: podSelector,
							},
						})
					}
				}

				// Configure namespace selector
				if len(policy.AllowTrafficFrom.Namespaces) > 0 {
					for _, nsSelector := range policy.AllowTrafficFrom.Namespaces {
						ingressRule.From = append(ingressRule.From, networkingv1.NetworkPolicyPeer{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: nsSelector,
							},
						})
					}
				}

				// Configure ports if specified
				if len(policy.AllowTrafficFrom.Ports) > 0 {
					for _, port := range policy.AllowTrafficFrom.Ports {
						portNumber := intstr.FromInt(port.Port)
						protocol := corev1.Protocol(port.Protocol)
						if protocol == "" {
							protocol = corev1.ProtocolTCP // Default to TCP if not specified
						}

						ingressRule.Ports = append(ingressRule.Ports, networkingv1.NetworkPolicyPort{
							Port:     &portNumber,
							Protocol: &protocol,
						})
					}
				}

				ingressRules = append(ingressRules, ingressRule)
			}

			// Configure egress rules if allowTrafficTo is specified
			if policy.AllowTrafficTo != nil {
				egressRule := networkingv1.NetworkPolicyEgressRule{}

				// Configure pod selector
				if len(policy.AllowTrafficTo.Pods) > 0 {
					for _, podSelector := range policy.AllowTrafficTo.Pods {
						egressRule.To = append(egressRule.To, networkingv1.NetworkPolicyPeer{
							PodSelector: &metav1.LabelSelector{
								MatchLabels: podSelector,
							},
						})
					}
				}

				// Configure namespace selector
				if len(policy.AllowTrafficTo.Namespaces) > 0 {
					for _, nsSelector := range policy.AllowTrafficTo.Namespaces {
						egressRule.To = append(egressRule.To, networkingv1.NetworkPolicyPeer{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: nsSelector,
							},
						})
					}
				}

				// Configure ports if specified
				if len(policy.AllowTrafficTo.Ports) > 0 {
					for _, port := range policy.AllowTrafficTo.Ports {
						portNumber := intstr.FromInt(port.Port)
						protocol := corev1.Protocol(port.Protocol)
						if protocol == "" {
							protocol = corev1.ProtocolTCP // Default to TCP if not specified
						}

						egressRule.Ports = append(egressRule.Ports, networkingv1.NetworkPolicyPort{
							Port:     &portNumber,
							Protocol: &protocol,
						})
					}
				}

				egressRules = append(egressRules, egressRule)
			}
		}

		// Set the combined rules in the network policy
		if len(ingressRules) > 0 {
			netpol.Spec.Ingress = ingressRules
		} else {
			// If no ingress rules specified, deny all ingress
			netpol.Spec.Ingress = []networkingv1.NetworkPolicyIngressRule{}
		}

		if len(egressRules) > 0 {
			netpol.Spec.Egress = egressRules
		} else {
			// If no egress rules specified, deny all egress
			netpol.Spec.Egress = []networkingv1.NetworkPolicyEgressRule{}
		}
	}

	// Set controller reference
	if err := controllerutil.SetControllerReference(uc, netpol, u.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference for NetworkPolicy: %w", err)
	}

	// Create or update the NetworkPolicy
	existing := &networkingv1.NetworkPolicy{}
	err := u.Get(ctx, client.ObjectKey{Name: netpol.Name, Namespace: uc.Name}, existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := u.Create(ctx, netpol); err != nil {
				return fmt.Errorf("failed to create NetworkPolicy: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get existing NetworkPolicy: %w", err)
		}
	} else {
		// Update existing policy
		existing.Spec = netpol.Spec
		if err := u.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update NetworkPolicy: %w", err)
		}
	}

	return nil
}
