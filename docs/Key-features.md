# Key Features

The **UserConfig Controller** is designed to simplify and automate the management of `UserConfig` resources in Kubernetes. Below are the key features of this product:

## 1. UserConfig Resource Management
- Watches for `UserConfig` resource events such as creation, modification, and deletion.
- Ensures seamless lifecycle management for user-specific configurations.

## 2. Automatic Namespace Creation
- Dynamically creates a dedicated namespace upon the creation of a `UserConfig` resource.
- Provides logical isolation for each `UserConfig`, ensuring secure and organized resource management.

## 3. Secure SealedSecret Integration
- Fetches encrypted secrets specified in the `UserConfig` manifest.
- Automatically creates and manages a `SealedSecret` resource within the namespace created for the `UserConfig`.
- Facilitates secure storage and usage of sensitive data.

## 4. ResourceQuota addition
- Attaches default resource quota to the namespace if not given by the user in the manifest.

## 5. Resource Cleanup on Deletion
- Ensures a clean environment by deleting all resources associated with a `UserConfig` resource when it is deleted.
- Removes the corresponding `SealedSecret` and namespace, preventing resource orphaning or clutter.

This feature set provides a robust foundation for managing user-specific configurations, namespaces, and secrets in a Kubernetes environment
