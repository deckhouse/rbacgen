## RBAC v2 Generator

The RBACv2 Generator is a tool designed to create RBACv2 cluster roles (manage/use) for modules based on their CRDs.

This tool reads and analyzes CRDs, and if they pass filters, it generates cluster roles in the ```templates/rbacv2``` directory. 
The type of generated role (use or manage) is determined by the resource’s subsystem:

• *Namespaced resources*: Use roles are generated.

• *Cluster-wide resources*: Manage roles are generated.

### Use
Use the following command to generate roles and docs:

```rbacgen generate . docs.yaml``` 

### Adding a Module

To add a module, create a file named module.yaml and rbac.yaml in the module’s directory.

The module.yaml file contains the information required to identify module.

The rbac.yaml file contains the information that is required to parse CRDs. 

### Spec examples

Below is an example for the ```deckhouse``` module. 
It includes the module name, module namespace, subsystems, and the path to the CRDs:

module.yaml
```yaml
name: deckhouse
weight: 2
namespace: d8-system
subsystems:
  - deckhouse
```

rbac.yaml:
```yaml
crds:
  - deckhouse-controller/crds/*.yaml
```

Even though this module does not have CRDs, manage roles will still be generated, 
as these roles are responsible for managing the module’s configuration.

By default, the tool generates roles only for resources in the ```deckhouse.io``` group. 
However, if a module provides additional resources in other groups, 
you can include them by specifying them in the spec(rbac.yaml):
```yaml
crds:
  - ee/modules/500-operator-trivy/crds/native/*.yaml
allowedResources:
  - group: aquasecurity.github.io
    resources:
      - all
```
