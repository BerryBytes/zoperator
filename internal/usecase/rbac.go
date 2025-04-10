package usecase

import (
	"context"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"
)

func (u *UserConfigUseCase) ReconcileRBAC(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error {
	if len(uc.Spec.Permissions.Resources) == 0 {
		return nil
	}

	if err := u.ReconcileRole(ctx, uc); err != nil {
		return fmt.Errorf("failed to reconcile role: %w", err)
	}

	if err := u.ReconcileServiceAccount(ctx, uc); err != nil {
		return fmt.Errorf("failed to reconcile service account: %w", err)
	}

	if err := u.ReconcileRoleBinding(ctx, uc); err != nil {
		return fmt.Errorf("failed to reconcile role binding: %w", err)
	}

	return nil
}

func (u *UserConfigUseCase) ReconcileRole(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uc.Name,
			Namespace: uc.Name,
		},
		Rules: []rbacv1.PolicyRule{},
	}

	// Map CRUD to Kubernetes verbs
	for _, perm := range uc.Spec.Permissions.Resources {
		rule := rbacv1.PolicyRule{
			APIGroups: getAPIGroup(perm.Resource),
			Resources: []string{mapActualResource(perm.Resource)},
			Verbs:     mapCRUDToVerbs(perm.Operation),
		}
		role.Rules = append(role.Rules, rule)
	}

	// Set controller reference
	if err := controllerutil.SetControllerReference(uc, role, u.Scheme); err != nil {
		return fmt.Errorf("failed to set role owner reference: %w", err)
	}

	// Create or update role
	if err := u.Create(ctx, role); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create role: %w", err)
		}

		// Get existing role for update
		existing := &rbacv1.Role{}
		if err := u.Get(ctx, client.ObjectKey{Name: uc.Name, Namespace: uc.Name}, existing); err != nil {
			return fmt.Errorf("failed to get existing role: %w", err)
		}

		existing.Rules = role.Rules
		if err := u.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update role: %w", err)
		}
	}

	return nil
}

func (u *UserConfigUseCase) ReconcileServiceAccount(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uc.Name,
			Namespace: uc.Name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":          "userconfig-operator",
				"userconfig.myoperator.01cloud.io/name": uc.Name,
			},
		},
	}

	// Set controller reference
	if err := controllerutil.SetControllerReference(uc, sa, u.Scheme); err != nil {
		return fmt.Errorf("failed to set serviceaccount owner reference: %w", err)
	}

	// Create or update ServiceAccount
	if err := u.Create(ctx, sa); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create serviceaccount: %w", err)
		}

		// Get existing ServiceAccount for update
		existing := &corev1.ServiceAccount{}
		if err := u.Get(ctx, client.ObjectKey{Name: uc.Name, Namespace: uc.Name}, existing); err != nil {
			return fmt.Errorf("failed to get existing serviceaccount: %w", err)
		}

		// Update labels if needed
		existing.Labels = sa.Labels
		if err := u.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update serviceaccount: %w", err)
		}
	}

	return nil
}

func (u *UserConfigUseCase) ReconcileRoleBinding(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error {
	subjects := []rbacv1.Subject{
		{
			Kind: "User",
			Name: uc.Name,
		},
		{
			Kind:      "ServiceAccount",
			Name:      uc.Name,
			Namespace: uc.Name,
		},
	}

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uc.Name,
			Namespace: uc.Name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":          "userconfig-operator",
				"userconfig.myoperator.01cloud.io/name": uc.Name,
			},
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     uc.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	// Set controller reference
	if err := controllerutil.SetControllerReference(uc, roleBinding, u.Scheme); err != nil {
		return fmt.Errorf("failed to set rolebinding owner reference: %w", err)
	}

	// Create or update RoleBinding
	if err := u.Create(ctx, roleBinding); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create rolebinding: %w", err)
		}

		// Get existing RoleBinding for update
		existing := &rbacv1.RoleBinding{}
		if err := u.Get(ctx, client.ObjectKey{Name: uc.Name, Namespace: uc.Name}, existing); err != nil {
			return fmt.Errorf("failed to get existing rolebinding: %w", err)
		}

		// Update subjects and roleRef
		existing.Subjects = subjects
		existing.RoleRef = roleBinding.RoleRef
		existing.Labels = roleBinding.Labels
		if err := u.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update rolebinding: %w", err)
		}
	}

	return nil
}

func getAPIGroup(resource string) []string {
	switch resource {
	case "deployments", "deployment", "replicasets", "replicaset", "statefulsets", "statefulset", "daemonsets", "daemonset", "scaledeployment", "scalereplicaset":
		return []string{"apps"}
	case "pods", "pod", "services", "service", "resourcequotas", "resourcequota", "limitranges", "limitrange", "secret", "secrets", "namespaces", "namespace", "serviceaccounts", "serviceaccount", "logs", "configmaps", "configmap", "persistentvolumeclaims", "persistentvolumeclaim", "persistentvolumes", "persistentvolume":
		return []string{""} // Core API group
	case "rolebindings", "rolebinding", "roles", "role":
		return []string{"rbac.authorization.k8s.io"}
	case "networkpolicies", "networkpolicy":
		return []string{"networking.k8s.io"}
	case "sealedsecrets", "sealedsecret":
		return []string{"bitnami.com"}
	case "ingresses", "ingress":
		return []string{"networking.k8s.io"}
	// Add any other specific mappings here
	default:
		// Don't use wildcard for unknown resources
		logger := log.FromContext(context.Background())
		logger.Info("Unknown resource type", "resource", resource)
		return []string{"apps"} // More conservative default
	}
}

func mapActualResource(resource string) string {
	switch resource {
	case "deployment", "deployments":
		return "deployments"
	case "replicaset", "replicasets":
		return "replicasets"
	case "statefulset", "statefulsets":
		return "statefulsets"
	case "daemonset", "daemonsets":
		return "daemonsets"
	case "pod", "pods":
		return "pods"
	case "service", "services":
		return "services"
	case "resourcequota", "resourcequotas":
		return "resourcequotas"
	case "limitrange", "limitranges":
		return "limitranges"
	case "secret", "secrets":
		return "secrets"
	case "namespace", "namespaces":
		return "namespaces"
	case "serviceaccount", "serviceaccounts":
		return "serviceaccounts"
	case "role", "roles":
		return "roles"
	case "rolebinding", "rolebindings":
		return "rolebindings"
	case "networkpolicy", "networkpolicies":
		return "networkpolicies"
	case "sealedsecret", "sealedsecrets":
		return "sealedsecrets"
	case "logs":
		return "pods/log"
	case "scaledeployment":
		return "deployments/scale"
	case "scalereplicaset":
		return "replicasets/scale"
	case "ingress", "ingresses":
		return "ingresses"
	default:
		return resource
	}
}

func mapCRUDToVerbs(operation string) []string {
	// Split operation string into individual characters
	ops := strings.Split(operation, "")

	// Remove duplicates using a map
	uniqueOps := make(map[string]bool)
	for _, op := range ops {
		uniqueOps[op] = true
	}

	// Convert unique operations to verbs
	verbMap := make(map[string]bool)
	for op := range uniqueOps {
		switch op {
		case "C":
			verbMap["create"] = true
		case "R":
			verbMap["get"] = true
			verbMap["list"] = true
			verbMap["watch"] = true
		case "U":
			verbMap["update"] = true
			verbMap["patch"] = true
		case "D":
			verbMap["delete"] = true
		}
	}

	// Convert verb map to slice
	verbs := make([]string, 0, len(verbMap))
	for verb := range verbMap {
		verbs = append(verbs, verb)
	}
	sort.Strings(verbs)

	// If no valid operations were found, provide read-only access as a fallback
	if len(verbs) == 0 {
		return []string{"get", "list", "watch"}
	}

	return verbs
}
