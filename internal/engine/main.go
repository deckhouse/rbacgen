package engine

import (
	"context"
	"fmt"
	rbacv1 "k8s.io/api/rbac/v1"
	"log"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"slices"
)

const SpecFile = "rbac.yaml"

const RolesBasePath = "templates/rbacv2"

type Spec struct {
	Module             string     `yaml:"module"`
	Namespace          string     `json:"namespace"`
	Subsystems         []string   `yaml:"subsystems"`
	CRDs               []string   `yaml:"crds"`
	AllowedResources   []Resource `yaml:"allowedResources"`
	ForbiddenResources []string   `yaml:"forbiddenResources"`
	path               string
}
type Resource struct {
	Group     string   `yaml:"group"`
	Resources []string `yaml:"resources"`
}

func WalkAndRender(ctx context.Context, dir, docsPath string) error {
	specs, err := walkSpecs(dir)
	if err != nil {
		return err
	}
	docs := doc{
		Subsystems: make(map[string]*subsystemDoc),
		Modules:    make(map[string]*moduleDoc),
	}
	for _, spec := range specs {
		moduleDocs, err := renderBySpec(ctx, spec)
		if err != nil {
			return err
		}
		docs.Modules[spec.Module] = moduleDocs
		docs.addSubsystemDoc(spec.Module, spec.Subsystems)
	}
	return docs.writeTo(docsPath)
}

func walkSpecs(dir string) (specs []*Spec, err error) {
	log.Printf("walking dir: %s\n", dir)
	err = walk(dir, true, []string{"internal", "testdata", "docs", ".github"}, func(path string) error {
		if filepath.Base(path) == SpecFile {
			spec, err := parseSpec(dir, filepath.Dir(path), path)
			if err != nil {
				return err
			}
			log.Printf("found spec for the '%s' module \n", spec.Module)
			specs = append(specs, spec)
			return nil
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	log.Printf("found %d specs\n", len(specs))
	return
}

func parseSpec(workDir, modulePath, specPath string) (*Spec, error) {
	log.Printf("parsing spec %s\n", specPath)
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}
	spec := new(Spec)
	if err = yaml.Unmarshal(raw, spec); err != nil {
		return nil, err
	}
	spec.path = modulePath
	for idx, crds := range spec.CRDs {
		spec.CRDs[idx] = filepath.Join(workDir, crds)
	}
	if spec.Namespace == "" {
		spec.Namespace = fmt.Sprintf("d8-%s", spec.Module)
	}
	return spec, nil
}

func renderBySpec(ctx context.Context, spec *Spec) (*moduleDoc, error) {
	manage, use, err := parseAndBuildRoles(ctx, spec)
	if err != nil {
		return nil, err
	}
	for _, role := range manage {
		if err = writeRole(spec.path, role); err != nil {
			return nil, err
		}
	}
	for _, role := range use {
		if err = writeRole(spec.path, role); err != nil {
			return nil, err
		}
	}
	return buildModuleDoc(spec.Namespace, spec.Subsystems, manage, use), err
}

func parseAndBuildRoles(ctx context.Context, spec *Spec) ([]*rbacv1.ClusterRole, []*rbacv1.ClusterRole, error) {
	var crds []string
	for _, dir := range spec.CRDs {
		tmp, err := filepath.Glob(dir)
		if err != nil {
			return nil, nil, err
		}
		crds = append(crds, tmp...)
	}
	parsed, err := newParser().parse(ctx, crds, func(group, resource string) bool {
		if slices.Contains(spec.ForbiddenResources, resource) {
			log.Printf("ignoring forbidden resource '%s/%s' for the '%s' module\n", group, resource, spec.Module)
			return false
		}
		if group == "deckhouse.io" {
			log.Printf("found the '%s/%s' resource for the '%s' module\n", group, resource, spec.Module)
			return true
		}
		for _, allowed := range spec.AllowedResources {
			if allowed.Group == group && (slices.Contains(allowed.Resources, resource) || allowed.Resources[0] == "all") {
				log.Printf("found the '%s/%s' resource for the '%s' module\n", group, resource, spec.Module)
				return true
			}
		}
		log.Printf("ignoring the '%s/%s' resource for the '%s' module\n", group, resource, spec.Module)
		return false
	})
	if err != nil {
		return nil, nil, err
	}
	manage, use := buildRoles(spec.Module, spec.Namespace, spec.Subsystems, parsed.ClusterCRDs, parsed.NamespacedCRDs)
	return manage, use, nil
}
