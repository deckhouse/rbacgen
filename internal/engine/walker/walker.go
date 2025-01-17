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

package walker

import (
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/deckhouse/rbacgen/internal/engine/models"
)

func WalkModules(dir string) ([]*models.Module, error) {
	var modules []*models.Module

	err := walk(dir, []string{"internal", "crds", "testdata", "docs", ".github"}, func(path string) error {
		if filepath.Base(path) == models.DefinitionFile {
			module, err := parseModule(dir, filepath.Dir(path))
			if err != nil {
				return err
			}
			if len(module.Definition.Subsystems) != 0 {
				modules = append(modules, module)
			}
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return modules, nil
}

// walk walks over specific directory
func walk(dir string, skippedDir []string, f func(path string) error) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if len(skippedDir) > 0 && slices.Contains(skippedDir, info.Name()) {
				return filepath.SkipDir
			}
		}

		return f(path)
	})
}

func parseModule(root, modulePath string) (*models.Module, error) {
	def, err := parseDefinition(filepath.Join(modulePath, models.DefinitionFile))
	if err != nil {
		return nil, err
	}

	spec, err := parseSpec(root, filepath.Join(modulePath, models.SpecFile))
	if err != nil {
		return nil, err
	}

	return &models.Module{Definition: def, Spec: spec, Path: modulePath}, nil
}

func parseDefinition(path string) (*models.Definition, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	def := new(models.Definition)
	if err = yaml.Unmarshal(raw, def); err != nil {
		return nil, err
	}

	return def, nil
}

func parseSpec(root, path string) (*models.Spec, error) {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return nil, nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	spec := new(models.Spec)
	if err = yaml.Unmarshal(raw, spec); err != nil {
		return nil, err
	}

	for idx, crd := range spec.CRDs {
		spec.CRDs[idx] = filepath.Join(root, crd)
	}

	return spec, nil
}
