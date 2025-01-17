// Copyright 2024 Flant JSC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package doc

import (
	"os"
	"sigs.k8s.io/yaml"
	"sort"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/deckhouse/rbacgen/internal/engine/models"
)

type Docs struct {
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

func New() *Docs {
	return &Docs{
		Subsystems: make(map[string]*subsystemDoc),
		Modules:    make(map[string]*moduleDoc),
	}
}

func (d *Docs) WriteTo(path string) error {
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

func (d *Docs) AddSubsystem(module *models.Module) {
	for _, subsystem := range module.Definition.Subsystems {
		if found, ok := d.Subsystems[subsystem]; ok {
			found.Modules = append(found.Modules, module.Definition.Name)
			if d.Modules[module.Definition.Name].Namespace != "" {
				found.namespacesSet.Insert(d.Modules[module.Definition.Name].Namespace)
			}
			d.Subsystems[subsystem] = found
		} else {
			docs := &subsystemDoc{Modules: []string{module.Definition.Name}, namespacesSet: sets.New[string]()}
			if d.Modules[module.Definition.Name].Namespace != "" {
				docs.namespacesSet.Insert(d.Modules[module.Definition.Name].Namespace)
			}
			d.Subsystems[subsystem] = docs
		}
	}
}

func (d *Docs) AddModule(module *models.Module, manageRoles, useRoles []*rbacv1.ClusterRole) {
	d.Modules[module.Definition.Name] = buildModuleDoc(module.Definition.Namespace, module.Definition.Subsystems, manageRoles, useRoles)
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
