package engine

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"os"
	"sigs.k8s.io/yaml"
	"sort"
)

type doc struct {
	Subsystems map[string]*subsystemDoc `json:"subsystems"`
	Modules    map[string]*moduleDoc    `json:"modules"`
}

type subsystemDoc struct {
	Modules       []string `json:"modules"`
	Namespaces    []string `json:"namespaces"`
	namespacesSet sets.Set[string]
}

type moduleDoc struct {
	Subsystems   []string        `json:"subsystems"`
	Capabilities capabilitiesDoc `json:"capabilities"`
	Namespace    string          `json:"namespace"`
}
type capabilitiesDoc struct {
	Manage []capabilityDoc `json:"manage"`
	Use    []capabilityDoc `json:"use"`
}
type capabilityDoc struct {
	Name  string              `json:"name"`
	Rules []rbacv1.PolicyRule `json:"rules"`
}

func (d *doc) addSubsystemDoc(module string, subsystems []string) {
	for _, subsystem := range subsystems {
		if found, ok := d.Subsystems[subsystem]; ok {
			found.Modules = append(found.Modules, module)
			if d.Modules[module].Namespace != "none" {
				found.namespacesSet.Insert(d.Modules[module].Namespace)
			}
			d.Subsystems[subsystem] = found
		} else {
			docs := &subsystemDoc{Modules: []string{module}, namespacesSet: sets.New[string]()}
			if d.Modules[module].Namespace != "none" {
				docs.namespacesSet.Insert(d.Modules[module].Namespace)
			}
			d.Subsystems[subsystem] = docs
		}
	}
}

func (d *doc) writeTo(path string) error {
	for key, docs := range d.Subsystems {
		if val, ok := d.Subsystems[key]; ok {
			val.Namespaces = docs.namespacesSet.UnsortedList()
			sort.Strings(val.Namespaces)
			d.Subsystems[key] = val
		}
	}

	marshaled, err := yaml.Marshal(d)
	if err != nil {
		return err
	}

	return os.WriteFile(path, marshaled, 0666)
}

func buildModuleDoc(namespace string, subsystems []string, manageRoles, useRoles []*rbacv1.ClusterRole) *moduleDoc {
	docs := &moduleDoc{Subsystems: subsystems, Namespace: namespace}
	for _, role := range manageRoles {
		docs.Capabilities.Manage = append(docs.Capabilities.Manage, capabilityDoc{
			Name:  role.Name,
			Rules: role.Rules,
		})
	}
	for _, role := range useRoles {
		docs.Capabilities.Use = append(docs.Capabilities.Use, capabilityDoc{
			Name:  role.Name,
			Rules: role.Rules,
		})
	}
	return docs
}
