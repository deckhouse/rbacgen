package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apimachineryYaml "k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"sort"
	"strings"
)

const (
	ScopeNamespaced = "Namespaced"
	ScopeCluster    = "Cluster"
)

const CustomResourceDefinitionKind = "CustomResourceDefinition"

type parser struct {
	buffer []byte
}

type filterFunc func(group, resource string) bool

type parsedCRDs struct {
	ClusterCRDs    map[string][]string
	NamespacedCRDs map[string][]string
}

func newParser() *parser {
	return &parser{buffer: make([]byte, 1*1024*1024)}
}

func (p *parser) parse(ctx context.Context, crds []string, filter filterFunc) (*parsedCRDs, error) {
	result := &parsedCRDs{
		ClusterCRDs:    make(map[string][]string),
		NamespacedCRDs: make(map[string][]string),
	}
	for _, crd := range crds {
		if strings.Contains(crd, "doc-") {
			continue
		}
		parsed, err := p.processFile(ctx, crd, filter)
		if err != nil {
			return nil, err
		}
		if len(parsed) != 0 {
			for _, parsedCRD := range parsed {
				if parsedCRD.Spec.Scope == ScopeCluster {
					result.ClusterCRDs[parsedCRD.Spec.Group] = append(result.ClusterCRDs[parsedCRD.Spec.Group], parsedCRD.Spec.Names.Plural)
				}
				if parsedCRD.Spec.Scope == ScopeNamespaced {
					result.NamespacedCRDs[parsedCRD.Spec.Group] = append(result.NamespacedCRDs[parsedCRD.Spec.Group], parsedCRD.Spec.Names.Plural)
				}
			}
		}
	}
	// to avoid flaky result
	sort.Strings(result.ClusterCRDs[ScopeCluster])
	sort.Strings(result.NamespacedCRDs[ScopeNamespaced])
	return result, nil
}
func (p *parser) processFile(ctx context.Context, path string, filter filterFunc) (crds []*apiextensionv1.CustomResourceDefinition, err error) {
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

		crd, err := p.parseCRD(ctx, bytes.NewReader(data), n, filter)
		if err != nil {
			return nil, err
		}
		if crd != nil {
			crds = append(crds, crd)
		}
	}
	return crds, nil
}
func (p *parser) parseCRD(_ context.Context, reader io.Reader, bufferSize int, filter filterFunc) (*apiextensionv1.CustomResourceDefinition, error) {
	var crd *apiextensionv1.CustomResourceDefinition
	if err := apimachineryYaml.NewYAMLOrJSONDecoder(reader, bufferSize).Decode(&crd); err != nil {
		return nil, err
	}
	// it could be a comment or some other peace of yaml file, skip it
	if crd == nil {
		return nil, nil
	}
	if crd.APIVersion != apiextensionv1.SchemeGroupVersion.String() && crd.Kind != CustomResourceDefinitionKind {
		return nil, fmt.Errorf("invalid CRD document apiversion/kind: '%s/%s'", crd.APIVersion, crd.Kind)
	}
	if filter(crd.Spec.Group, crd.Spec.Names.Plural) {
		return crd, nil
	}
	return nil, nil
}
