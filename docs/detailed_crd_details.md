# UserConfig CustomResourceDefinition

This document provides a detailed description of the `UserConfig` Custom Resource Definition (CRD) in the `myoperator.01cloud.io` API group.

## Overview

The `UserConfig` CRD allows you to define and manage user configurations within a Kubernetes cluster. It provides a structured way to manage user identities, permissions, resource quotas, network policies, and more at the cluster level.

## API Version and Kind

```yaml
apiVersion: myoperator.01cloud.io/v1alpha1
kind: UserConfig
```

## Resource Definition

### Metadata

Standard Kubernetes metadata applies to the `UserConfig` resource. The CRD is cluster-scoped.

### Spec Fields

The `UserConfig` specification consists of the following main sections:

#### Identity

The `identity` section is required and contains user identification and group membership details.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | User's unique identifier. Must be DNS-compatible, 3-63 characters, lowercase alphanumeric with hyphens (cannot start or end with hyphen). |
| `contact` | string | Yes | User's email address for communication. Must be a valid email format. |
| `groups` | array of strings | No | User's group memberships with predefined roles. |
| `labels` | array of strings | No | Optional additional tags for user classification. |

#### Permissions

The `permissions` section is required and defines access levels for specific Kubernetes resources.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `resources` | array of ResourcePermission | No | List of resource permissions granted to the user. |

**ResourcePermission:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `resource` | string | Yes | Type of Kubernetes resource. Allowed values: `deployment`, `service`, `secret`, `pods`, `configmap`, `ingress`, `persistentvolumeclaim`, `logs`, `scaledeployment`, `scalereplicaset`. |
| `operation` | string | Yes | Allowed operations on the resource. Can be a combination of C(create), R(read), U(update), D(delete) or "*" for full access. Maximum length: 4 characters. |

#### ResourceQuota

The `resourceQuota` section defines resource quota configuration for the namespace.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `cpu` | string | No | CPU quota for the namespace. Must match pattern `^([0-9]+)([mKMGTP]*i?)$`. |
| `memory` | string | No | Memory quota for the namespace. Must match pattern `^([0-9]+)([mKMGTP]*i?)$`. |
| `ephemeral-storage` | string | No | Ephemeral storage quota. Must match pattern `^([0-9]+)([mKMGTP]*i?)$`. |
| `pods` | string | No | Maximum number of pods. Must be numeric. |
| `services` | string | No | Maximum number of services. Must be numeric. |
| `services.nodeports` | string | No | Maximum number of node port services. Must be numeric. |
| `services.loadbalancers` | string | No | Maximum number of load balancer services. Must be numeric. |
| `secrets` | string | No | Maximum number of secrets. Must be numeric. |
| `persistentvolumeclaims` | string | No | Maximum number of persistent volume claims. Must be numeric. |
| `replicationcontrollers` | string | No | Maximum number of replication controllers. Must be numeric. |
| `requests.cpu` | string | No | Request quotas for CPU. Must match pattern `^([0-9]+)([mKMGTP]*i?)$`. |
| `requests.memory` | string | No | Request quotas for memory. Must match pattern `^([0-9]+)([mKMGTP]*i?)$`. |
| `requests.storage` | string | No | Request quotas for storage. Must match pattern `^([0-9]+)([mKMGTP]*i?)$`. |
| `requests.ephemeral-storage` | string | No | Request quotas for ephemeral storage. Must match pattern `^([0-9]+)([mKMGTP]*i?)$`. |
| `requests.configmaps` | string | No | Maximum number of config maps. Must be numeric. |
| `limits.cpu` | string | No | Limit quotas for CPU. Must match pattern `^([0-9]+)([mKMGTP]*i?)$`. |
| `limits.memory` | string | No | Limit quotas for memory. Must match pattern `^([0-9]+)([mKMGTP]*i?)$`. |
| `limits.ephemeral-storage` | string | No | Limit quotas for ephemeral storage. Must match pattern `^([0-9]+)([mKMGTP]*i?)$`. |

#### LimitRange

The `limitRange` section defines the limits of resources usable by containers.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `limits` | array of LimitRangeLimit | No | Defines the limit range of resources usable by containers. |

**LimitRangeLimit:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Type of resource. Can be either "Container" or "Pod". |
| `min` | object | No | Smallest allowed resource a container can request or limit. |
| `max` | object | No | Maximum allowed resource a container can request or limit. |
| `default` | object | No | Default resource cap assigned to the container if not assigned any. |
| `defaultRequest` | object | No | Default usable resource allocated to container can request if not assigned any. |

Each of these objects (`min`, `max`, `default`, `defaultRequest`) can contain:

| Field | Type | Description |
|-------|------|-------------|
| `cpu` | string | CPU resource specification. Sample values: 100m, 1, 1.5 |
| `memory` | string | Memory resource specification. Sample values: 100Mi, 1Gi, 1.5Gi |

#### NetworkPolicy

The `networkPolicy` section defines the network policy configuration.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `allowTrafficFrom` | object | No | Specifies allowed traffic sources. |
| `allowTrafficTo` | object | No | Specifies allowed traffic destinations. |

Both `allowTrafficFrom` and `allowTrafficTo` can contain:

| Field | Type | Description |
|-------|------|-------------|
| `namespaces` | array of objects | Specifies allowed namespaces. Each object contains key-value pairs. |
| `pods` | array of objects | Specifies allowed pods. Each object contains key-value pairs. |
| `ports` | array of NetworkPolicyPort | Specifies allowed network ports. |

**NetworkPolicyPort:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `port` | integer | Yes | Port number for network policy, between 1-65535. |
| `protocol` | string | No | Protocol for network traffic. Allowed values: TCP (default), UDP, SCTP. |

#### Secrets

The `secrets` section defines the secrets configuration.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Name of the secret. Must be DNS-compatible. |
| `type` | string | Yes | Type of secret. Can be either "sealed" or "external". |
| `sealedSecret` | object | No | Used to define sealed secrets. |
| `externalSecret` | object | No | Used to define external secrets from other providers (upcoming feature). |

**SealedSecret:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `encryptedData` | object | Yes | Contains the encrypted data for the sealed secret. Key-value pairs. |

**ExternalSecret:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | Yes | External secret provider. Allowed values: aws, gcp, azure, vault. |
| `endpoint` | string | Yes | Endpoint for the external secret provider. Must be a valid URL. |
| `secretPath` | string | Yes | Path to the secret in the external provider. |
| `credentials` | object | Yes | Credentials for accessing the external secret provider. |

**Credentials:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `accessKey` | string | Yes | Access key for the external secret provider. |
| `secretKey` | string | Yes | Secret key for the external secret provider. |

#### ServiceAccounts

The `serviceAccounts` section defines the service accounts configuration.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Name of the service account. Must be DNS-compatible. |
| `imagePullSecrets` | array of strings | No | Image pull secrets for the service account. |

### Status Fields

The `UserConfig` status represents the observed state of the resource:

| Field | Type | Description |
|-------|------|-------------|
| `state` | string | Current state of the UserConfig. Values: "Pending", "Active", "Error". |
| `lastUpdated` | date-time | Timestamp of when the status was last updated. |
| `conditions` | array of Condition | Detailed conditions of the resource. |

**Condition:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Type of condition in CamelCase. |
| `status` | string | Yes | Status of the condition. Values: "True", "False", "Unknown". |
| `lastTransitionTime` | date-time | Yes | Last time the condition transitioned from one status to another. |
| `reason` | string | Yes | Programmatic identifier indicating the reason for the condition's last transition. |
| `message` | string | Yes | Human-readable message indicating details about the transition. |
| `observedGeneration` | integer | No | The .metadata.generation that the condition was set based upon. |

## Additional Columns

The CRD defines additional printer columns for the `kubectl get` command:

| Column Name | Source | Type |
|-------------|--------|------|
| Status | .status.state | string |
| Age | .metadata.creationTimestamp | date |
| Username | .spec.identity.username | string |

## Example Usage

```yaml
apiVersion: myoperator.01cloud.io/v1alpha1
kind: UserConfig
metadata:
  name: example-user
spec:
  identity:
    username: example-user
    contact: user@example.com
    groups:
      - developers
    labels:
      - tier-1

  permissions:
    resources:
      - resource: deployment
        operation: CRUD
      - resource: service
        operation: CR

  resourceQuota:
    cpu: 4
    memory: 8Gi
    pods: 10

  limitRange:
    limits:
      - type: Container
        min:
          cpu: 100m
          memory: 100Mi
        max:
          cpu: 2
          memory: 4Gi
        default:
          cpu: 200m
          memory: 512Mi
        defaultRequest:
          cpu: 100m
          memory: 256Mi

  networkPolicy:
    - allowTrafficFrom:
        namespaces:
          - kubernetes.io/metadata.name: frontend-namespace
        pods:
          - app: frontend
        ports:
          - port: 80
            protocol: TCP
    - allowTrafficTo:
        namespaces:
          - kubernetes.io/metadata.name: backend-namespace
        pods:
          - app: database
        ports:
          - port: 5432
            protocol: TCP

  secrets:
    - name: example-sealed-secret
      type: sealed
      sealedSecret:
        encryptedData:
          username: ENCRYPTED_VALUE_HERE
          password: ENCRYPTED_VALUE_HERE

  serviceAccounts:
    - name: example-service-account
      imagePullSecrets:
        - docker-registry-secret
```

## Notes

1. The resource quota patterns follow Kubernetes resource notation:
   - CPU: "100m" (100 millicores), "1" (1 core), etc.
   - Memory: "100Mi" (100 Mebibytes), "1Gi" (1 Gibibyte), etc.

2. For operations in permissions, note that if using `kubectl apply`, the Create action requires GET permission as well.

3. External secrets are marked as an upcoming feature and not yet implemented.

4. Username must be DNS-compatible: lowercase alphanumeric characters with hyphens, 3-63 characters in length.

5. This CRD is cluster-scoped, not namespace-scoped.
