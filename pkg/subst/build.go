package subst

import (
	"path/filepath"

	"github.com/buttahtoast/subst/pkg/config"
)

type Build struct {
	root          string
	Paths         []string
	Manifests     []manifest
	Substitutions Substitutions
	keys          []string
	cfg           config.Configuration
}

type Substitutions struct {
	Subst struct {
		Env     map[string]string           `yaml:"env"`
		Vars    map[interface{}]interface{} `yaml:"vars"`
		Secrets map[interface{}]interface{} `yaml:"secrets"`
	} `yaml:"subst"`
}

func New(config config.Configuration) (*Build, error) {
	result := &Build{
		root: config.RootDirectory,
		cfg:  config,
		keys: config.EjsonKey,
	}

	// Kustomize Build
	err := result.build()
	if err != nil {
		return nil, err
	}

	return result, err
}

func (b *Build) build() error {

	err := b.kustomizePaths(b.root)
	if err != nil {
		return err
	}

	err = b.kustomizeBuild()
	if err != nil {
		return err
	}

	err = b.walkpaths(b.varsWalk)
	if err != nil {
		return err
	}

	// Ejson
	err = b.loadEjsonKeys()
	if err != nil {
		return err
	}

	err = b.walkpaths(b.ejsonWalk)
	if err != nil {
		return err
	}

	return nil

}

func (b *Build) walkpaths(fn filepath.WalkFunc) error {
	for path := range b.Paths {
		err := filepath.Walk(b.Paths[path], fn)
		if err != nil {
			return err
		}
	}
	return nil
}
