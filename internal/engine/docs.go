package engine

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"os"
	"sigs.k8s.io/yaml"
	"sort"
)

type doc struct {
	Scopes  map[string]*scopeDoc  `json:"scopes"`
	Modules map[string]*moduleDoc `json:"modules"`
}

type scopeDoc struct {
	Modules       []string `json:"modules"`
	Namespaces    []string `json:"namespaces"`
	namespacesSet sets.Set[string]
}

type moduleDoc struct {
	Scopes       []string        `json:"scopes"`
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

func (d *doc) addScopeDoc(module string, scopes []string) {
	for _, scope := range scopes {
		if found, ok := d.Scopes[scope]; ok {
			found.Modules = append(found.Modules, module)
			if d.Modules[module].Namespace != "none" {
				found.namespacesSet.Insert(d.Modules[module].Namespace)
			}
			d.Scopes[scope] = found
		} else {
			docs := &scopeDoc{Modules: []string{module}, namespacesSet: sets.New[string]()}
			if d.Modules[module].Namespace != "none" {
				docs.namespacesSet.Insert(d.Modules[module].Namespace)
			}
			d.Scopes[scope] = docs
		}
	}
}

func (d *doc) writeTo(path string) error {
	for key, docs := range d.Scopes {
		if val, ok := d.Scopes[key]; ok {
			val.Namespaces = docs.namespacesSet.UnsortedList()
			sort.Strings(val.Namespaces)
			d.Scopes[key] = val
		}
	}

	marshaled, err := yaml.Marshal(d)
	if err != nil {
		return err
	}

	return os.WriteFile(path, marshaled, 0666)
}

func buildModuleDoc(namespace string, scopes []string, manageRoles, useRoles []*rbacv1.ClusterRole) *moduleDoc {
	docs := &moduleDoc{Scopes: scopes, Namespace: namespace}
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
