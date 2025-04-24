package usecase

import (
	"context"
	"fmt"
	"os"
	"strings"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"
)

func (u *UserConfigUseCase) GenerateAndSaveKubeconfig(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error {
	// Get the ServiceAccount
	sa := &corev1.ServiceAccount{}
	if err := u.Get(ctx, client.ObjectKey{
		Namespace: uc.Name,
		Name:      uc.Name,
	}, sa); err != nil {
		return fmt.Errorf("failed to get ServiceAccount: %w", err)
	}

	// Retrieve the Kind API server address
	cluster, clusterName, err := getKindServerAddress()
	if err != nil {
		return fmt.Errorf("failed to get Kind server address: %w", err)
	}

	serverCA, err := getKindCA()
	if err != nil {
		return fmt.Errorf("failed to get Kind CA: %w", err)
	}

	// Try to get the configuration, with fallback options
	var config *rest.Config

	// First try getting the in-cluster config
	config, err = rest.InClusterConfig()
	if err != nil {
		// If that fails, try getting the default config (works for local development)
		config, err = ctrl.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to get any valid kubernetes config: %w", err)
		}
	}

	// Create clientset for core API calls
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	pointedValue := int64(86400 * 365)

	// Create the TokenRequest object
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences: []string{},
			// Set expiration to 1 year
			ExpirationSeconds: &pointedValue,
		},
	}

	// Create the token request through the API
	tokenResponse, err := clientset.CoreV1().ServiceAccounts(uc.Name).CreateToken(
		ctx,
		uc.Name,
		tokenRequest,
		metav1.CreateOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to create token: %w", err)
	}

	if tokenResponse.Status.Token == "" {
		return fmt.Errorf("received empty token from API")
	}

	// Determine cluster name and context name
	contextName := fmt.Sprintf("%s-context", uc.Name)

	// Create kubeconfig structure
	kubeconfig := clientcmdapi.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*clientcmdapi.Cluster{
			clusterName: {
				Server:                   cluster.Server,
				CertificateAuthorityData: []byte(serverCA),
				InsecureSkipTLSVerify:    config.Insecure,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			contextName: {
				Cluster:   clusterName,
				AuthInfo:  uc.Name,
				Namespace: uc.Name,
			},
		},
		CurrentContext: contextName,
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			uc.Name: {
				Token: tokenResponse.Status.Token,
			},
		},
	}

	// Convert kubeconfig to bytes
	kubeconfigBytes, err := clientcmd.Write(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to serialize kubeconfig: %w", err)
	}

	// Create Secret object for storing kubeconfig
	kubeconfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kubeconfig", uc.Name),
			Namespace: uc.Name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":          "userconfig-operator",
				"userconfig.myoperator.01cloud.io/name": uc.Name,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"kubeconfig": kubeconfigBytes,
		},
	}

	// Set controller reference
	if err := controllerutil.SetControllerReference(uc, kubeconfigSecret, u.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Create or update the Secret
	err = u.Create(ctx, kubeconfigSecret)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create kubeconfig secret: %w", err)
		}
		// Update existing secret
		existing := &corev1.Secret{}
		if err := u.Get(ctx, client.ObjectKey{Name: kubeconfigSecret.Name, Namespace: uc.Name}, existing); err != nil {
			return fmt.Errorf("failed to get existing secret: %w", err)
		}
		existing.Data = kubeconfigSecret.Data
		existing.Labels = kubeconfigSecret.Labels
		if err := u.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update kubeconfig secret: %w", err)
		}
	}

	return nil
}

func getKindServerAddress() (*clientcmdapi.Cluster, string, error) {
	kubeconfigPath := "/config"
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil && strings.Contains(err.Error(), "no such file or directory") {
		kubeconfigPath = os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			return nil, "", fmt.Errorf("KUBECONFIG environment variable is not set")
		}
		config, err = clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to load kubeconfig from KUBECONFIG: %w", err)
		}
	} else if err != nil {
		return nil, "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Assuming the current context points to the Kind cluster
	currentContext := config.CurrentContext
	if currentContext == "" {
		return nil, "", fmt.Errorf("no current context found in kubeconfig")
	}

	context, exists := config.Contexts[currentContext]
	if !exists {
		return nil, "", fmt.Errorf("context %s not found", currentContext)
	}

	cluster, exists := config.Clusters[context.Cluster]
	if !exists {
		return nil, "", fmt.Errorf("cluster %s not found", context.Cluster)
	}

	return cluster, cluster.Server, nil
}

func getKindCA() (string, error) {
	kubeconfigPath := "/config"
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil && strings.Contains(err.Error(), "no such file or directory") {
		kubeconfigPath = os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			return "", fmt.Errorf("KUBECONFIG environment variable is not set")
		}
		config, err = clientcmd.LoadFromFile(kubeconfigPath)
		if err != nil {
			return "", fmt.Errorf("failed to load kubeconfig from KUBECONFIG: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Assuming the current context points to the Kind cluster
	currentContext := config.CurrentContext
	if currentContext == "" {
		return "", fmt.Errorf("no current context found in kubeconfig")
	}

	context, exists := config.Contexts[currentContext]
	if !exists {
		return "", fmt.Errorf("context %s not found", currentContext)
	}

	cluster, exists := config.Clusters[context.Cluster]
	if !exists {
		return "", fmt.Errorf("cluster %s not found", context.Cluster)
	}

	return string(cluster.CertificateAuthorityData), nil
}
