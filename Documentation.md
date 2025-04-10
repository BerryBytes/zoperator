## Overview
The zoperator is a kubernetes operator built suing Kubebuilder that manages tenant configs and their resources during and after creation of the tenants. It introduces `UserConfig` that defines tenant-specific settings and automatically provisions required resources based on the configuration.

## Features
- Custom Resource Definition (CRD) for managing user configurations
- Automated namespace creation for each tenant
- Resource lifecycle management with finalizers
- Status tracking for UserConfig resources
- Comprehensive validation rules for configuration fields
- Support for various resource permissions and configurations
- Integration with Sealed Secrets for secure secret management

### Required RBAC Permissions

The operator requires the following RBAC permissions:

```yaml
# UserConfig resources
- apiGroups: ["myoperator.01cloud.io"]
  resources: ["userconfigs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["myoperator.01cloud.io"]
  resources: ["userconfigs/status"]
  verbs: ["get", "update", "patch"]
- apiGroups: ["myoperator.01cloud.io"]
  resources: ["userconfigs/finalizers"]
  verbs: ["update"]
# Sealed Secrets
- apiGroups: ["bitnami.com"]
  resources: ["sealedsecrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

### Resource Management

The operator manages the following resources for each UserConfig:

1. **Namespace**
   - Created with name format: `<userconfig-name>-namespace`
   - Labeled with:
     - `app.kubernetes.io/managed-by: userconfig-operator`
     - `userconfig.myoperator.01cloud.io/name: <userconfig-name>`
   - Automatically deleted when UserConfig is deleted

2. **Sealed Secrets**
   - Managed within the tenant namespace
   - Supports creation and updates
   - Automatically cleaned up during tenant deletion
   - Requires proper encryption using kubeseal

### Status Management

The UserConfig resource maintains the following states:
- `Active`: Resources successfully reconciled
- `Error`: Issues encountered during reconciliation

### Sample UserConfig with Sealed Secrets

```yaml
apiVersion: myoperator.01cloud.io/v1alpha1
kind: UserConfig
metadata:
  name: tenant-1
spec:
  identity:
    username: "tenant-1"
    contact: "tenant@example.com"
  secrets:
    - name: my-secret
      type: sealed
      sealedSecret:
        encryptedData:
          key1: <encrypted-value>
          key2: <encrypted-value>
```

To create a sealed secret:
1. Create a regular secret:
   ```bash
   kubectl create secret generic my-secret --from-literal=key1=value1 --dry-run=client -o yaml > secret.yaml
   ```
2. Encrypt it using kubeseal:
   ```bash
   kubeseal --format yaml < secret.yaml > sealed-secret.yaml
   ```
3. Copy the encryptedData section to your UserConfig manifest

## Development Procedures

### Prerequisites for kubebuilder v4.3.1
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- kubeseal CLI tool (for sealed secrets)


### Setting up the Environment

1. **Install Kubebuilder:**

    ```bash
    curl -L -o kubebuilder "https://go.kubebuilder.io/dl/v4.3.1/$(go env GOOS)/$(go env GOARCH)"

    chmod +x kubebuilder && sudo mv kubebuilder /usr/local/bin/

    ```

2. **Install kubeseal CLI:**

    ```bash
    # For Linux
    wget https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.24.5/kubeseal-0.24.5-linux-amd64.tar.gz
    tar -xvzf kubeseal-0.24.5-linux-amd64.tar.gz
    sudo install -m 755 kubeseal /usr/local/bin/kubeseal

    # For MacOS
    brew install kubeseal
    ```

3. **Create a KInd cluster:**

    ```bash
    kind create cluster --name operator-test
    ```

4. **Install Sealed Secrets Controller:**

    ```bash
    kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.27.3/controller.yaml
    ```

5. **Make Manifests and CRD:**
    ```bash
    # creat the CRD
    make manifests
    ```

    The project layout looks like this:
    ```
    .
    ├── api/
    │   └── v1alpha1/
    │       ├── userconfig_types.go    # CRD type definitions
    │       └── zz_generated.deepcopy.go
    ├── config/
    │   ├── crd/                      # CRD manifests
    │   ├── rbac/                     # RBAC configurations
    │   └── samples/                  # Sample CR manifests
    └── controllers/
        └── userconfig_controller.go  # Main reconciliation logic
    ```

6. **Install the CRDs into the cluster:**
    ```bash
    make install
    ```

7. **Run the operator:**

    ```bash
    make run
    ```

8. **Apply below sample manifest:**

    ```yaml
    apiVersion: myoperator.01cloud.io/v1alpha1
    kind: UserConfig
    metadata:
      name: tenant-1
    spec:
      identity:
        username: "tenant-1"
        contact: "tenant@example.com"
        groups:
        - viewer
      permissions:
        resources:
        - resource: deployment
          level: editor
      clusterRoles:
        - viewer
    ```

9. **Verify the resources created:**
    ```bash
    kubectl get userconfig tenant-1 -o yaml
    kubectl get ns tenant-1-namespace
    ```

### Troubleshooting:
#### Common issues and solutions:

1. CRD Not Found

    - Ensure CRDs are installed: `make install`
    - Check CRD status: `kubectl get crds | grep userconfig`


2. Namespace Creation Failed

    - Check operator logs: `kubectl logs -n zoperator-system <pod-name>`
    - Verify RBAC permissions: `kubectl auth can-i create namespace --as system:serviceaccount:zoperator-system:zoperator-controller-manager`


3. Status Updates Failed

    - Ensure status subresource is enabled in CRD
    - Check for validation errors in the status update

4. Sealed Secrets Issues

    - Verify Sealed Secrets controller is running: `kubectl get pods -n kube-system | grep sealed-secrets`
    - Check if kubeseal can connect to the controller: `kubeseal --fetch-cert`
    - Verify secret encryption: Try encrypting a test secret with kubeseal

## R&D on progress:
- Include SealedSecrets operator
- Shared informer between this two operator.

## UserConfig CRD Specification

### Overview
The UserConfig CRD (Custom Resource Definition) provides a declarative way to manage tenant configurations and their associated resources in a Kubernetes cluster.

### Structure and Fields

#### 1. Identity Configuration
```yaml
spec:
  identity:
    username: "tenant-1"      # Required, DNS-compatible (3-63 chars)
    contact: "user@org.com"   # Required, valid email
    groups:                   # Optional, max 6 groups
    - viewer
    - developer              # Allowed: viewer, developer, tester, admin, operations, security
    labels:                  # Optional classification
    - team-a
```

#### 2. Resource Permissions
```yaml
spec:
  permissions:
    resources:
    - resource: deployment   # Supported: deployment, service, secret, pods, configmap, ingress
      level: editor         # Levels: viewer, editor, admin, CRUD, R, RU
  clusterRoles:            # Default: ["viewer"]
  - viewer                 # Available: viewer, developer, admin, tester
```

#### 3. Secret Management
Supports two types of secrets:

##### Sealed Secrets:
```yaml
spec:
  secrets:
  - name: "db-creds"
    type: sealed
    sealedSecret:
      encryptedData:
        username: "encrypted-value"
        password: "encrypted-value"
```

##### External Secrets:
```yaml
spec:
  secrets:
  - name: "api-keys"
    type: external
    externalSecret:
      provider: aws         # Supported: aws, gcp, azure, vault
      endpoint: "https://..."
      credentials:
        accessKey: "key"
        secretKey: "secret"
      secretPath: "path/to/secret"
```

#### 4. Resource Controls

##### Resource Quotas:
```yaml
spec:
  resourceQuota:
    "requests.cpu": "1"
    "requests.memory": "1Gi"
    "limits.cpu": "2"
    "limits.memory": "2Gi"
```

##### Limit Ranges:
```yaml
spec:
  limitRange:
    limits:
    - type: Container
      max:
        cpu: "2"
        memory: "2Gi"
      min:
        cpu: "100m"
        memory: "100Mi"
      default:
        cpu: "500m"
        memory: "500Mi"
```

#### 5. Network Policies
```yaml
spec:
  networkPolicy:
  - allowTrafficFrom:
      pods:
        - app: "frontend"
      namespaces:
        - environment: "prod"
  - allowTrafficTo:
      pods:
        - app: "database"
```

#### 6. Service Accounts
```yaml
spec:
  serviceAccounts:
  - name: "app-sa"
    imagePullSecrets:
    - "registry-creds"
```

### Status and Conditions

The UserConfig maintains status information:

```yaml
status:
  state: Active    # Possible values: Pending, Active, Error
  lastUpdated: "2024-03-21T10:00:00Z"
  conditions:
  - type: Ready
    status: "True"
    reason: ResourcesCreated
    message: "All resources successfully reconciled"
```

### Validation Rules

1. **Identity**:
   - Username: Must be DNS-compatible, 3-63 characters
   - Groups: Maximum 6 groups from predefined list
   - Contact: Valid email format required

2. **Permissions**:
   - Resources must be from supported list
   - Access levels must match predefined values
   - ClusterRoles default to "viewer" if not specified

3. **Secrets**:
   - Secret names must be DNS-compatible
   - Type must be either "sealed" or "external"
   - Provider-specific fields required for external secrets

4. **Resource Controls**:
   - Resource values must use valid Kubernetes quantity format
   - LimitRange type must be "Container"
   - Valid resource metrics (cpu, memory) required

### Best Practices

1. **Resource Management**:
   - Always set resource quotas for production tenants
   - Configure appropriate limit ranges to prevent resource abuse
   - Use default limits for predictable behavior

2. **Security**:
   - Use minimum required permissions
   - Implement network policies for isolation
   - Rotate secrets regularly
   - Use sealed secrets for sensitive information

3. **Naming and Organization**:
   - Use consistent naming conventions
   - Apply meaningful labels
   - Group related configurations

4. **Monitoring**:
   - Watch for status changes
   - Monitor resource usage against quotas
   - Track permission changes

### Integration Patterns

1. **External Systems**:
   - Secret management systems (AWS Secrets Manager, Vault)
   - Identity providers
   - Monitoring systems

2. **Other Operators**:
   - Sealed Secrets Operator
   - External Secrets Operator
   - Cert Manager
   - Ingress Controllers
