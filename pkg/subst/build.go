package subst

import (
	"path/filepath"

	"github.com/buttahtoast/subst/pkg/config"
	"github.com/geofffranks/spruce"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/resmap"
)

type Build struct {
	root          string
	Paths         []string
	Manifests     []map[interface{}]interface{}
	Substitutions Substitutions
	keys          []string
	cfg           config.Configuration
}

type Substitutions struct {
	Subst map[interface{}]interface{} `yaml:"subst"`
}

func (s *Substitutions) tointerface() (map[interface{}]interface{}, error) {
	tmp := make(map[interface{}]interface{})
	yml, err := yaml.Marshal(s)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yml, tmp)
	return tmp, err
}

func New(config config.Configuration) (build *Build, err error) {
	result := &Build{
		root: config.RootDirectory,
		cfg:  config,
		keys: config.EjsonKey,
	}

	// Gather all releveant paths
	err = result.kustomizePaths(result.root)
	if err != nil {
		return nil, err
	}

	// Gather all available substitutions
	err = result.gatherSubstituions()
	if err != nil {
		return nil, err
	}

	return result, err
}

func (b *Build) gatherSubstituions() (err error) {

	// Read Vars Files
	err = b.walkpaths(b.varsWalk)
	if err != nil {
		return err
	}

	err = b.runEjson()
	if err != nil {
		return err
	}

	return nil
}

func (b *Build) Build() error {
	var manifests resmap.ResMap

	manifests, err := b.kustomizeBuild()
	if err != nil {
		return err
	}

	// Load Substitutions
	substs, err := b.Substitutions.tointerface()
	if err != nil {
		return err
	}

	// Run Spruce
	for _, manifest := range manifests.Resources() {

		// Load Single Manifest
		m, _ := manifest.AsYAML()
		var str map[interface{}]interface{}
		err := yaml.Unmarshal(m, &str)
		if err != nil {
			return err
		}

		substManifest, err := spruce.Merge(str, substs)
		if err != nil {
			return err
		}

		// Evaluate with Spruce
		evaluator := &spruce.Evaluator{
			Tree:     substManifest,
			SkipEval: b.cfg.SkipEvaluation,
		}

		// Evaluate Tree and prune subst
		err = evaluator.Run([]string{"subst"}, nil)
		if err != nil {
			return err
		}
		b.Manifests = append(b.Manifests, evaluator.Tree)
	}

	return nil
}

// Generic funtion to walk all paths and run a function on each file
func (b *Build) walkpaths(fn filepath.WalkFunc) error {
	for path := range b.Paths {
		err := filepath.Walk(b.Paths[path], fn)
		if err != nil {
			return err
		}
	}
	return nil
}
