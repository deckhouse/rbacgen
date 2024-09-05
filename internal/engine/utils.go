package engine

import (
	"fmt"
	rbacv1 "k8s.io/api/rbac/v1"
	"log"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"slices"
	"strings"
)

// walk walks over specific directory
func walk(dir string, skipDir bool, skippedDir []string, f func(path string) error) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if skipDir && info.IsDir() {
			if skippedDir != nil && len(skippedDir) > 0 && slices.Contains(skippedDir, info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if f != nil {
			return f(path)
		}
		return nil
	})
}

func writeRole(path string, role *rbacv1.ClusterRole) error {
	kind := KindUse
	if strings.Contains(role.Name, ":"+KindManage+":") {
		kind = KindManage
	}

	name := VerbView
	if strings.HasSuffix(role.Name, ":"+VerbEdit) {
		name = VerbEdit
	}

	if err := os.MkdirAll(filepath.Join(path, RolesBasePath, kind), 0755); err != nil {
		return err
	}

	marshaled, err := yaml.Marshal(role)
	if err != nil {
		return err
	}

	log.Printf("writing role %s to %s\n", role.Name, filepath.Join(path, RolesBasePath, kind))
	return os.WriteFile(filepath.Join(path, RolesBasePath, kind, fmt.Sprintf("%s.yaml", name)), marshaled, 0644)
}
