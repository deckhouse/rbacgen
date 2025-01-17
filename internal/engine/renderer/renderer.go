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

package renderer

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"slices"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	apimachineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/deckhouse/rbacgen/internal/engine/doc"
	"github.com/deckhouse/rbacgen/internal/engine/models"
	"github.com/deckhouse/rbacgen/internal/engine/parser"
)

const (
	moduleDeckhouse = "deckhouse"

	verbView = "view"
	verbEdit = "edit"

	kindManage = "manage"
	kindUse    = "use"

	roleManager = "manager"
	roleViewer  = "viewer"

	templatesPath = "templates/rbacv2"
)

var (
	verbsView = []string{"get", "list", "watch"}
	verbsEdit = []string{"create", "update", "patch", "delete", "deletecollection"}

	subsystemTemplate = "rbac.deckhouse.io/aggregate-to-%s-as"
)

func Render(ctx context.Context, modules []*models.Module) (*doc.Docs, error) {
	docs := doc.New()
	for _, module := range modules {
		if err := render(ctx, module, docs); err != nil {
			return nil, err
		}
	}
	return docs, nil
}

func render(ctx context.Context, module *models.Module, docs *doc.Docs) error {
	parsed, err := parser.Parse(ctx, module)
	if err != nil {
		return err
	}

	manage, use := buildRoles(module, parsed.Cluster, parsed.Namespaced)

	for _, role := range manage {
		if err = writeRole(module.Path, role); err != nil {
			return err
		}
	}

	for _, role := range use {
		if err = writeRole(module.Path, role); err != nil {
			return err
		}
	}

	docs.AddModule(module, manage, use)
	docs.AddSubsystem(module)

	return nil
}

func buildRoles(module *models.Module, manageResources, useResources map[string][]string) ([]*rbacv1.ClusterRole, []*rbacv1.ClusterRole) {
	var useViewRules, useEditRules, manageViewRules, manageEditRules []rbacv1.PolicyRule

	// rules for manage roles
	for group, resources := range manageResources {
		manageViewRules = append(manageViewRules, rbacv1.PolicyRule{
			APIGroups: []string{group},
			Resources: resources,
			Verbs:     verbsView,
		})
		manageEditRules = append(manageEditRules, rbacv1.PolicyRule{
			APIGroups: []string{group},
			Resources: resources,
			Verbs:     verbsEdit,
		})
	}

	slices.SortFunc(manageViewRules, func(r1, r2 rbacv1.PolicyRule) int {
		return cmp.Compare(r1.APIGroups[0], r2.APIGroups[0])
	})

	slices.SortFunc(useEditRules, func(r1, r2 rbacv1.PolicyRule) int {
		return cmp.Compare(r1.APIGroups[0], r2.APIGroups[0])
	})

	// rules for use roles
	for group, resources := range useResources {
		useViewRules = append(useViewRules, rbacv1.PolicyRule{
			APIGroups: []string{group},
			Resources: resources,
			Verbs:     verbsView,
		})
		useEditRules = append(useEditRules, rbacv1.PolicyRule{
			APIGroups: []string{group},
			Resources: resources,
			Verbs:     verbsEdit,
		})
	}

	//deckhouse can manage all module configs
	if module.Definition.Name != moduleDeckhouse {
		manageViewRules = append(manageViewRules, rbacv1.PolicyRule{
			APIGroups:     []string{"deckhouse.io"},
			Resources:     []string{"moduleconfigs"},
			ResourceNames: []string{module.Definition.Name},
			Verbs:         verbsView,
		})
		manageEditRules = append(manageEditRules, rbacv1.PolicyRule{
			APIGroups:     []string{"deckhouse.io"},
			Resources:     []string{"moduleconfigs"},
			ResourceNames: []string{module.Definition.Name},
			Verbs:         []string{"create", "update", "patch", "delete"},
		})
	}

	var manageRoles = []*rbacv1.ClusterRole{
		buildRole(module, roleViewer, kindManage, verbView, manageViewRules),
		buildRole(module, roleManager, kindManage, verbEdit, manageEditRules),
	}

	var useRoles []*rbacv1.ClusterRole
	if len(useEditRules) > 0 && len(useViewRules) > 0 {
		useRoles = append(useRoles, buildRole(module, roleViewer, kindUse, verbView, useViewRules))
		useRoles = append(useRoles, buildRole(module, roleManager, kindUse, verbEdit, useEditRules))
	}

	return manageRoles, useRoles
}

func buildRole(module *models.Module, rbacRole, rbacKind, rbacVerb string, rules []rbacv1.PolicyRule) *rbacv1.ClusterRole {
	var role *rbacv1.ClusterRole
	if rbacKind == kindUse {
		role = &rbacv1.ClusterRole{
			TypeMeta: apimachineryv1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRole",
			},
			ObjectMeta: apimachineryv1.ObjectMeta{
				Name: fmt.Sprintf("d8:%s:capability:module:%s:%s", rbacKind, module.Definition.Name, rbacVerb),
				Labels: map[string]string{
					"heritage":               "deckhouse",
					"module":                 module.Definition.Name,
					"rbac.deckhouse.io/kind": rbacKind,
					"rbac.deckhouse.io/aggregate-to-kubernetes-as": rbacRole,
				},
			},
			Rules: rules,
		}
	}
	if rbacKind == kindManage {
		role = &rbacv1.ClusterRole{
			TypeMeta: apimachineryv1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRole",
			},
			ObjectMeta: apimachineryv1.ObjectMeta{
				Name: fmt.Sprintf("d8:%s:permission:module:%s:%s", rbacKind, module.Definition.Name, rbacVerb),
				Labels: map[string]string{
					"heritage":                "deckhouse",
					"module":                  module.Definition.Name,
					"rbac.deckhouse.io/kind":  rbacKind,
					"rbac.deckhouse.io/level": "module",
				},
			},
			Rules: rules,
		}
		for _, subsystem := range module.Definition.Subsystems {
			role.ObjectMeta.Labels[fmt.Sprintf(subsystemTemplate, subsystem)] = rbacRole
		}
		if module.Definition.Namespace != "" {
			role.ObjectMeta.Labels["rbac.deckhouse.io/namespace"] = module.Definition.Namespace
		}
	}
	return role
}

func writeRole(path string, role *rbacv1.ClusterRole) error {
	kind := kindUse
	if strings.Contains(role.Name, ":"+kindManage+":") {
		kind = kindManage
	}

	name := verbView
	if strings.HasSuffix(role.Name, ":"+verbEdit) {
		name = verbEdit
	}

	if err := os.MkdirAll(filepath.Join(path, templatesPath, kind), 0755); err != nil {
		return err
	}

	marshaled, err := yaml.Marshal(role)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(path, templatesPath, kind, fmt.Sprintf("%s.yaml", name)), marshaled, 0644)
}
