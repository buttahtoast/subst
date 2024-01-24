package kustomize

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type Kustomize struct {
	Root  string
	Paths []string
	Build resmap.ResMap
}

func NewKustomize(root string) (*Kustomize, error) {
	k := &Kustomize{Root: root}
	if err := k.build(); err != nil {
		return nil, err
	}
	if err := k.paths(root); err != nil {
		return nil, err
	}
	if err := k.addPath(root); err != nil {
		return nil, err
	}
	return k, nil
}

var kustomizeBuildMutex sync.Mutex

func (k *Kustomize) addPath(path string) error {
	p, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for _, v := range k.Paths {
		if v == p {
			return nil
		}
	}
	k.Paths = append(k.Paths, p)
	return nil
}

func (k *Kustomize) paths(path string) error {
	path = convertPath(path)
	kz, err := kustomizeFile(path)
	if err != nil {
		return err
	}

	for _, patch := range kz.Patches {
		if err := k.addPath(filepath.Join(path, filepath.Dir(patch.Path))); err != nil {
			return err
		}
	}

	for _, resource := range kz.Resources {
		p := filepath.Join(path, resource)
		file, err := os.Stat(p)
		if err != nil {
			return err
		}
		if file.IsDir() {
			p = convertPath(p)
			if err := k.paths(p); err != nil {
				return err
			}
			if err := k.addPath(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k *Kustomize) Walk(fn func(path string, f fs.FileInfo) error) error {
	for _, path := range k.Paths {
		files, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range files {
			file, err := entry.Info()
			if err != nil {
				return err
			}
			if err := fn(path, file); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k *Kustomize) build() (err error) {
	fs := filesys.MakeFsOnDisk()

	kustomizeBuildMutex.Lock()
	defer kustomizeBuildMutex.Unlock()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from kustomize build panic: %v", r)
		}
	}()

	buildOptions := &krusty.Options{
		LoadRestrictions: kustypes.LoadRestrictionsNone,
		PluginConfig:     kustypes.EnabledPluginConfig(kustypes.BuiltinPluginLoadingOptions(kustypes.PluginRestrictionsNone)),
	}

	b := krusty.MakeKustomizer(buildOptions)

	k.Build, err = b.Run(fs, k.Root)
	return err
}
