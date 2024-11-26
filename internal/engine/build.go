package engine

import (
	"fmt"
	rbacv1 "k8s.io/api/rbac/v1"
	apimachineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	VerbsView = []string{"get", "list", "watch"}
	VerbsEdit = []string{"create", "update", "patch", "delete", "deletecollection"}
)

var subsystemTemplate = "rbac.deckhouse.io/aggregate-to-%s-as"

const (
	VerbView = "view"
	VerbEdit = "edit"
)

const (
	KindManage = "manage"
	KindUse    = "use"
)

const (
	RoleManager = "manager"
	RoleViewer  = "viewer"
)

func buildRoles(module, namespace string, subsystems []string, manageResources, useResources map[string][]string) ([]*rbacv1.ClusterRole, []*rbacv1.ClusterRole) {
	var useViewRules, useEditRules, manageViewRules, manageEditRules []rbacv1.PolicyRule

	// rules for manage roles
	for group, resources := range manageResources {
		manageViewRules = append(manageViewRules, rbacv1.PolicyRule{
			APIGroups: []string{group},
			Resources: resources,
			Verbs:     VerbsView,
		})
		manageEditRules = append(manageEditRules, rbacv1.PolicyRule{
			APIGroups: []string{group},
			Resources: resources,
			Verbs:     VerbsEdit,
		})
	}

	// rules for use roles
	for group, resources := range useResources {
		useViewRules = append(useViewRules, rbacv1.PolicyRule{
			APIGroups: []string{group},
			Resources: resources,
			Verbs:     VerbsView,
		})
		useEditRules = append(useEditRules, rbacv1.PolicyRule{
			APIGroups: []string{group},
			Resources: resources,
			Verbs:     VerbsEdit,
		})
	}

	//deckhouse can manage all module configs
	if module != "deckhouse" {
		manageViewRules = append(manageViewRules, rbacv1.PolicyRule{
			APIGroups:     []string{"deckhouse.io"},
			Resources:     []string{"moduleconfigs"},
			ResourceNames: []string{module},
			Verbs:         VerbsView,
		})
		manageEditRules = append(manageEditRules, rbacv1.PolicyRule{
			APIGroups:     []string{"deckhouse.io"},
			Resources:     []string{"moduleconfigs"},
			ResourceNames: []string{module},
			Verbs:         []string{"create", "update", "patch", "delete"},
		})
	}

	var manageRoles = []*rbacv1.ClusterRole{
		buildRole(module, namespace, subsystems, RoleViewer, KindManage, VerbView, manageViewRules),
		buildRole(module, namespace, subsystems, RoleManager, KindManage, VerbEdit, manageEditRules),
	}

	var useRoles []*rbacv1.ClusterRole
	if len(useEditRules) > 0 && len(useViewRules) > 0 {
		useRoles = append(useRoles, buildRole(module, namespace, subsystems, RoleViewer, KindUse, VerbView, useViewRules))
		useRoles = append(useRoles, buildRole(module, namespace, subsystems, RoleManager, KindUse, VerbEdit, useEditRules))
	}

	return manageRoles, useRoles
}
func buildRole(module, namespace string, subsystems []string, rbacRole, rbacKind, rbacVerb string, rules []rbacv1.PolicyRule) *rbacv1.ClusterRole {
	var role *rbacv1.ClusterRole
	if rbacKind == KindUse {
		role = &rbacv1.ClusterRole{
			TypeMeta: apimachineryv1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRole",
			},
			ObjectMeta: apimachineryv1.ObjectMeta{
				Name: fmt.Sprintf("d8:%s:capability:module:%s:%s", rbacKind, module, rbacVerb),
				Labels: map[string]string{
					"heritage":                            "deckhouse",
					"module":                              module,
					"rbac.deckhouse.io/kind":              rbacKind,
					"rbac.deckhouse.io/aggregate-to-role": rbacRole,
				},
			},
			Rules: rules,
		}
	}
	if rbacKind == KindManage {
		role = &rbacv1.ClusterRole{
			TypeMeta: apimachineryv1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "ClusterRole",
			},
			ObjectMeta: apimachineryv1.ObjectMeta{
				Name: fmt.Sprintf("d8:%s:permission:module:%s:%s", rbacKind, module, rbacVerb),
				Labels: map[string]string{
					"heritage":                "deckhouse",
					"module":                  module,
					"rbac.deckhouse.io/kind":  rbacKind,
					"rbac.deckhouse.io/level": "module",
				},
			},
			Rules: rules,
		}
		for _, subsystem := range subsystems {
			role.ObjectMeta.Labels[fmt.Sprintf(subsystemTemplate, subsystem)] = rbacRole
		}
		if namespace != "none" {
			role.ObjectMeta.Labels["rbac.deckhouse.io/namespace"] = namespace
		}
	}
	return role
}
