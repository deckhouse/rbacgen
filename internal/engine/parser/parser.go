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

package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apimachineryYaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/deckhouse/rbacgen/internal/engine/models"
)

const (
	customResourceDefinitionKind = "CustomResourceDefinition"

	scopeNamespaced = "Namespaced"
	scopeCluster    = "Cluster"

	deckhouseGroup = "deckhouse.io"

	allResources = "all"
)

type parser struct {
	buffer []byte
}

type ParsedCRDs struct {
	Cluster    map[string][]string
	Namespaced map[string][]string
}

func Parse(ctx context.Context, module *models.Module) (*ParsedCRDs, error) {
	result := &ParsedCRDs{
		Cluster:    make(map[string][]string),
		Namespaced: make(map[string][]string),
	}

	if module.Spec == nil {
		return result, nil
	}

	var crds []string
	for _, dir := range module.Spec.CRDs {
		tmp, err := filepath.Glob(dir)
		if err != nil {
			return nil, err
		}
		crds = append(crds, tmp...)
	}

	for _, crd := range crds {
		if strings.Contains(crd, "doc-") {
			continue
		}
		p := &parser{buffer: make([]byte, 1*1024*1024)}
		parsed, err := p.processFile(ctx, crd, module.Spec)
		if err != nil {
			return nil, err
		}
		if len(parsed) != 0 {
			for _, parsedCRD := range parsed {
				if parsedCRD.Spec.Scope == scopeCluster {
					result.Cluster[parsedCRD.Spec.Group] = append(result.Cluster[parsedCRD.Spec.Group], parsedCRD.Spec.Names.Plural)
				}
				if parsedCRD.Spec.Scope == scopeNamespaced {
					result.Namespaced[parsedCRD.Spec.Group] = append(result.Namespaced[parsedCRD.Spec.Group], parsedCRD.Spec.Names.Plural)
				}
			}
		}
	}

	return result, nil
}

func (p *parser) processFile(ctx context.Context, path string, spec *models.Spec) (crds []*apiextensionv1.CustomResourceDefinition, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := apimachineryYaml.NewDocumentDecoder(file)
	for {
		n, err := reader.Read(p.buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		data := p.buffer[:n]
		// some empty yaml document, or empty string before separator
		if len(data) == 0 {
			continue
		}

		crd, err := p.parseCRD(ctx, spec, bytes.NewReader(data), n)
		if err != nil {
			return nil, err
		}
		if crd != nil {
			crds = append(crds, crd)
		}
	}
	return crds, nil
}

func (p *parser) parseCRD(_ context.Context, spec *models.Spec, reader io.Reader, bufferSize int) (*apiextensionv1.CustomResourceDefinition, error) {
	crd := new(apiextensionv1.CustomResourceDefinition)
	if err := apimachineryYaml.NewYAMLOrJSONDecoder(reader, bufferSize).Decode(&crd); err != nil {
		return nil, err
	}

	// it could be a comment or some other peace of yaml file, skip it
	if crd == nil {
		return nil, nil
	}

	if crd.APIVersion != apiextensionv1.SchemeGroupVersion.String() && crd.Kind != customResourceDefinitionKind {
		return nil, fmt.Errorf("invalid CRD('%s/%s')", crd.APIVersion, crd.Kind)
	}

	if strings.Contains(crd.Name, "templates.gatekeeper.sh") {
		println()
	}

	if filter(spec, crd.Spec.Group, crd.Spec.Names.Plural) {
		return crd, nil
	}

	return nil, nil
}

func filter(spec *models.Spec, group, resource string) bool {
	if slices.Contains(spec.ForbiddenResources, resource) {
		return false
	}

	if group == deckhouseGroup {
		return true
	}

	for _, allowed := range spec.AllowedResources {
		if allowed.Group == group && (slices.Contains(allowed.Resources, resource) || allowed.Resources[0] == allResources) {
			return true
		}
	}

	return false
}
