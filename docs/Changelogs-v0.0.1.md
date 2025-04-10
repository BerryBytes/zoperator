# Changelogs

## [0.0.1] - 2025-01-16
### Initial Release

### Features

- **UserConfig Resource Management**
  The controller watches for the creation, modification, and deletion of `UserConfig` resources.

- **Automatic Namespace Creation**
  - On creation of a `UserConfig` resource, the controller dynamically provisions a dedicated namespace associated with the resource.
  - Ensures isolation and organization for each user configuration.

- **SealedSecret Resource Management**
  - Fetches encrypted secrets from the `UserConfig` manifest and creates a `SealedSecret` resource.
  - Places the `SealedSecret` within the namespace created for the `UserConfig` resource, maintaining secure storage and retrieval of secrets.

- **Resource Cleanup on Deletion**
  - Automatically deletes all resources associated with a `UserConfig` resource when it is removed.
  - This includes the corresponding `SealedSecret` and the namespace, ensuring a clean and consistent environment.

### Notes
- This version lays the groundwork for a robust user configuration management system and emphasizes secure and efficient resource handling.
- Future updates will focus on enhancements to scalability, and additional Rsource management.
