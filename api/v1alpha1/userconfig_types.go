package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition types for UserConfig status
const (
	// Ready condition indicates the UserConfig is fully reconciled
	ReadyCondition string = "Ready"
	// Reconciling condition indicates the UserConfig is being processed
	ReconcilingCondition string = "Reconciling"
	// Pending condition indicates the UserConfig is pending for some Reason
	PendingCondition string = "Pending"
	// Error condition indicates there was an error during reconciliation
	ErrorCondition  string = "Error"
	UserConfigReady string = "Ready"
)

// Identity defines the user identity configuration
type Identity struct {
	// Username is the user's unique identifier, must be DNS-compatible.
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=63
	Username string `json:"username"`

	// Groups represent user's group membership with predefined roles
	// +kubebuilder:validation:Type=array
	// +kubebuilder:validation:Items:type=string
	// +kubebuilder:validation:Items:enum=viewer;developer;tester;admin;operations;security
	// +optional
	Groups []string `json:"groups,omitempty"`

	// Contact is the user's email address for communication.
	// +kubebuilder:validation:Pattern="^[a-zA-Z._%+-]+@[a-zA-Z.-]+\\.[a-zA-Z]{2,}$"
	Contact string `json:"contact"`

	// Labels are optional additional tags for user classification.
	// +optional
	Labels []string `json:"labels,omitempty"`
}

// ResourcePermission defines access level for specific Kubernetes resources
type ResourcePermission struct {
	// Resource specifies the type of Kubernetes resource.
	// +kubebuilder:validation:Enum=deployment;service;secret;pods;configmap;ingress;persistentvolumeclaim;logs;scaledeployment;scalereplicaset;persistentvolume;
	Resource string `json:"resource"`

	// Operation specifies the allowed operations on the resource
	// Can be a combination of C(create), R(read), U(update), D(delete)
	// or "*" for full access
	// NOTE: If using kubectl apply, Create action requires GET permission
	// https://spacelift.io/blog/kubectl-apply-vs-create
	// +kubebuilder:validation:Pattern=^[CRUD*]+$
	// +kubebuilder:validation:MaxLength=4
	Operation string `json:"operation"`
}

// Permissions defines the overall permission configuration
type Permissions struct {
	// Resources is a list of resource permissions granted to the user.
	// +kubebuilder:validation:MinItems=1
	Resources []ResourcePermission `json:"resources"`
}

// Credentials defines the credentials for external secrets
type Credentials struct {
	// AccessKey is the access key for the external secret provider
	AccessKey string `json:"accessKey"`
	// SecretKey is the secret key for the external secret provider
	SecretKey string `json:"secretKey"`
}

// SealedSecret defines configuration for a Sealed Secret
type SealedSecret struct {
	// EncryptedData contains the encrypted data for the sealed secret
	EncryptedData map[string]string `json:"encryptedData"`
}

// ExternalSecret defines configuration for an External Secret
type ExternalSecret struct {
	// Provider specifies the external secret provider
	// +kubebuilder:validation:Enum=aws;gcp;azure;vault
	Provider string `json:"provider"`
	// Endpoint specifies the endpoint for the external secret provider
	// +kubebuilder:validation:Pattern=`^https?://[a-zA-Z0-9._-]+(:[0-9]+)?(/.*)?$`
	Endpoint string `json:"endpoint"`
	// Credentials contains the credentials for accessing the external secret provider
	Credentials Credentials `json:"credentials"`
	// SecretPath specifies the path to the secret in the external Provider
	// +kubebuilder:validation:Pattern=^/([a-zA-Z0-9._-]+/)*[a-zA-Z0-9._-]+$
	SecretPath string `json:"secretPath"`
}

// Secret defines the configuration for a secret
type Secret struct {
	// Name is the name of the secret
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	Name string `json:"name"`
	// Type specifies the type of secret
	// +kubebuilder:validation:Enum=sealed;external
	Type string `json:"type"`
	// SealedSecret is used to define sealed secrets
	SealedSecret *SealedSecret `json:"sealedSecret,omitempty"`
	// ExternalSecret is used to define external secrets from other providers
	// NOTE: THIS IS AN UPCOMING FEATURE. SO IT IS NOT YET IMPLEMENTED
	ExternalSecret *ExternalSecret `json:"externalSecret,omitempty"`
}

// ServiceAccount defines the service account configuration
type ServiceAccount struct {
	// Name is the name of the service account
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
	Name string `json:"name"`
	// ImagePullSecrets specifies the image pull secrets for the service account
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`
}

// Resources defines resource limits
type Resources struct {
	// CPU specifies the CPU resource limit and must be a valid CPU resource quantity
	// sample values: 100m, 1, 1.5
	// +kubebuilder:validation:Pattern=^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$
	CPU string `json:"cpu,omitempty"`
	// Memory specifies the memory resource limit and must be a valid memory resource quantity
	// sample values: 100Mi, 1Gi, 1.5Gi
	// +kubebuilder:validation:Pattern=^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$
	Memory string `json:"memory,omitempty"`
}

// ResourceQuota defines the resource quotas for a namespace
type ResourceQuota struct {
	// CPU quota for the namespace
	// +optional
	// +kubebuilder:validation:Pattern=^([0-9]+)([mKMGTP]*i?)$
	CPU string `json:"cpu,omitempty"`

	// Memory quota for the namespace
	// +optional
	// +kubebuilder:validation:Pattern=^([0-9]+)([mKMGTP]*i?)$
	Memory string `json:"memory,omitempty"`

	// Ephemeral storage quota
	// +optional
	// +kubebuilder:validation:Pattern=^([0-9]+)([mKMGTP]*i?)$
	EphemeralStorage string `json:"ephemeral-storage,omitempty"`

	// Request quotas for CPU
	// +optional
	// +kubebuilder:validation:Pattern=^([0-9]+)([mKMGTP]*i?)$
	RequestsCPU string `json:"requests.cpu,omitempty"`

	// Request quotas for memory
	// +optional
	// +kubebuilder:validation:Pattern=^([0-9]+)([mKMGTP]*i?)$
	RequestsMemory string `json:"requests.memory,omitempty"`

	// Request quotas for storage
	// +optional
	// +kubebuilder:validation:Pattern=^([0-9]+)([mKMGTP]*i?)$
	RequestsStorage string `json:"requests.storage,omitempty"`

	// Request quotas for ephemeral storage
	// +optional
	// +kubebuilder:validation:Pattern=^([0-9]+)([mKMGTP]*i?)$
	RequestsEphemeralStorage string `json:"requests.ephemeral-storage,omitempty"`

	// Limit quotas for CPU
	// +optional
	// +kubebuilder:validation:Pattern=^([0-9]+)([mKMGTP]*i?)$
	LimitsCPU string `json:"limits.cpu,omitempty"`

	// Limit quotas for memory
	// +optional
	// +kubebuilder:validation:Pattern=^([0-9]+)([mKMGTP]*i?)$
	LimitsMemory string `json:"limits.memory,omitempty"`

	// Limit quotas for ephemeral storage
	// +optional
	// +kubebuilder:validation:Pattern=^([0-9]+)([mKMGTP]*i?)$
	LimitsEphemeralStorage string `json:"limits.ephemeral-storage,omitempty"`

	// Maximum number of pods
	// +optional
	// +kubebuilder:validation:Pattern=^[0-9]+$
	Pods string `json:"pods,omitempty"`

	// Maximum number of services
	// +optional
	// +kubebuilder:validation:Pattern=^[0-9]+$
	Services string `json:"services,omitempty"`

	// Maximum number of replication controllers
	// +optional
	// +kubebuilder:validation:Pattern=^[0-9]+$
	ReplicationControllers string `json:"replicationcontrollers,omitempty"`

	// Maximum number of secrets
	// +optional
	// +kubebuilder:validation:Pattern=^[0-9]+$
	Secrets string `json:"secrets,omitempty"`

	// Maximum number of config maps
	// +optional
	// +kubebuilder:validation:Pattern=^[0-9]+$
	ConfigMaps string `json:"requests.configmaps,omitempty"`

	// Maximum number of persistent volume claims
	// +optional
	// +kubebuilder:validation:Pattern=^[0-9]+$
	PersistentVolumeClaims string `json:"persistentvolumeclaims,omitempty"`

	// Maximum number of node port services
	// +optional
	// +kubebuilder:validation:Pattern=^[0-9]+$
	ServicesNodePorts string `json:"services.nodeports,omitempty"`

	// Maximum number of load balancer services
	// +optional
	// +kubebuilder:validation:Pattern=^[0-9]+$
	ServicesLoadBalancers string `json:"services.loadbalancers,omitempty"`
}

// LimitRangeLimit defines the limit range of resource usable by container
type LimitRangeLimit struct {
	// Type specifies the type of resource, which can be either "Container" or "Pod", and in case of Pod Default resources are not set as they are not applicable
	// +kubebuilder:validation:Enum=Container;Pod
	Type string `json:"type"`

	// Maximum allowed resource a container can request or limit. Cannot be assigned below this.
	// +optional
	Max *Resources `json:"max,omitempty"`

	// Smallest allowed resource a container can request or limit. Cannot be assigned above this
	// +optional
	Min *Resources `json:"min,omitempty"`

	// default resource cap assigned to the container if not assigned any
	// +optional
	Default *Resources `json:"default,omitempty"`

	// default usable resource allocated to container can request if not assigned any
	// +optional
	DefaultRequest *Resources `json:"defaultRequest,omitempty"`
}

// LimitRange defines the limit of resource usable by container
type LimitRange struct {
	Limits []LimitRangeLimit `json:"limits,omitempty"`
}

// NetworkPolicyPeer defines allowed traffic sources or destinations
type NetworkPolicyPeer struct {
	// Pods specifies the allowed pods
	Pods []map[string]string `json:"pods,omitempty"`
	// Namespaces specifies the allowed namespaces
	Namespaces []map[string]string `json:"namespaces,omitempty"`

	// Ports specifies the allowed network ports
	// +optional
	Ports []NetworkPolicyPort `json:"ports,omitempty"`
}

// NetworkPolicy defines the network policy configuration
type NetworkPolicy struct {
	// AllowTrafficFrom specifies the allowed traffic sources
	// Example:
	// - allowTrafficFrom:
	//     namespaces:
	//       - kubernetes.io/metadata.name: frontend-namespace  # Allow traffic from namespace-a
	//     pods:
	//       - app: frontend  # Allow traffic from pods labeled 'frontend'
	AllowTrafficFrom *NetworkPolicyPeer `json:"allowTrafficFrom,omitempty"`

	// AllowTrafficTo specifies the allowed traffic destinations
	// Example:
	// - allowTrafficTo:
	//     namespaces:
	//       - kubernetes.io/metadata.name: test-user-namespace # Allow traffic to namespace-b
	//     pods:
	//       - app: backend  # Allow traffic to pods labeled 'backend'
	//    ports:
	//       - port: 80
	AllowTrafficTo *NetworkPolicyPeer `json:"allowTrafficTo,omitempty"`
}

// NetworkPolicyPort defines a port and protocol for network policies
type NetworkPolicyPort struct {
	// Port number for network policy, through which traffic is allowed
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int `json:"port"`

	// Protocol for network traffic (defaults to TCP)
	// +kubebuilder:validation:Enum=TCP;UDP;SCTP
	// +kubebuilder:default="TCP"
	Protocol string `json:"protocol,omitempty"`
}

// UserConfigSpec defines the desired state of UserConfig
type UserConfigSpec struct {
	// Identity contains the user identification and group membership details
	// +kubebuilder:validation:Required
	Identity Identity `json:"identity"`

	// Permissions defines the access level for specific Kubernetes resources
	// +kubebuilder:validation:Required
	Permissions Permissions `json:"permissions"`

	// Secrets defines the secrets configuration
	// +optional
	Secrets []Secret `json:"secrets,omitempty"`

	// ServiceAccounts defines the service accounts configuration
	// +optional
	ServiceAccounts []ServiceAccount `json:"serviceAccounts,omitempty"`

	// ResourceQuotas defines the resource quota configuration to the namespace
	// +optional
	ResourceQuotas *ResourceQuota `json:"resourceQuota,omitempty"`

	// LimitRange defines the limits of resource usable by the container.
	// +optional
	LimitRange *LimitRange `json:"limitRange,omitempty"`

	// NetworkPolicy defines the network policy configuration
	// +optional
	NetworkPolicy []NetworkPolicy `json:"networkPolicy,omitempty"`
}

// UserConfigStatus defines the observed state of UserConfig
type UserConfigStatus struct {
	// State represents the current state of the UserConfig
	// +kubebuilder:validation:Enum=Pending;Active;Error
	State string `json:"state,omitempty"`

	// +kubebuilder:validation:Format=date-time
	LastUpdated string `json:"lastUpdated,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Username",type="string",JSONPath=".spec.identity.username"
// +kubebuilder:resource:shortName=ucfg
// +kubebuilder:resource:scope=Cluster
type UserConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserConfigSpec   `json:"spec,omitempty"`
	Status UserConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UserConfigList contains a list of UserConfig
type UserConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UserConfig{}, &UserConfigList{})
}
