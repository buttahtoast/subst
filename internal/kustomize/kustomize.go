package kustomize

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type Kustomize struct {
	// Root Directory
	Root string
	// Paths resolved from kustomization files
	Paths []string
	// Kustomization outputs
	Build resmap.ResMap
}

func NewKustomize(root string) (k *Kustomize, err error) {
	k = &Kustomize{
		Root: root,
	}
	err = k.build()
	if err != nil {
		return nil, err
	}
	err = k.paths(root)
	if err != nil {
		return nil, err
	}
	err = k.addPath(root)
	if err != nil {
		return nil, err
	}

	return k, nil
}

// buildMutex protects against kustomize concurrent map read/write panic
var kustomizeBuildMutex sync.Mutex

// Add a new path (must be below kustomize root)
func (k *Kustomize) addPath(path string) error {
	p, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	//p, _, err := securePaths(k.Root, path)
	//if err != nil {
	//	logrus.Error(err)
	//} else {
	//	logrus.Debug("adding path: ", path)
	//	k.Paths = append(k.Paths, path)
	//}

	// if path is not already in the list, add it
	for _, v := range k.Paths {
		if v == p {
			return nil
		}
	}

	k.Paths = append(k.Paths, p)
	return nil
}

// Resolve all paths from the kustomization file
func (k *Kustomize) paths(path string) (err error) {
	path = convertPath(path)
	kz, err := kustomizeFile(path)
	if err != nil {
		return err
	}

	// Add Patch Locations to paths
	for _, patch := range kz.Patches {
		k.addPath(fmt.Sprintf("%v%v", path, filepath.Dir(patch.Path)))
	}

	for _, resource := range kz.Resources {
		p := fmt.Sprintf("%v%v", path, resource)
		file, _ := os.Stat(p)
		if file.IsDir() {
			p = convertPath(p)
			err := k.paths(p)
			if err != nil {
				return err
			}

			// Add Parent path after recursion (Has higher priority)
			k.addPath(p)
		}
	}
	return nil
}

// Generic funtion to walk all paths and run a function on each file
func (k *Kustomize) Walk(fn func(path string, f fs.FileInfo) error) error {
	for path := range k.Paths {
		buildDir, err := ioutil.ReadDir(k.Paths[path])

		if err != nil && err != fs.SkipDir {
			return err
		}

		for _, file := range buildDir {
			err = fn(k.Paths[path], file)
			if err != nil && err != fs.SkipDir {
				return err
			}
		}
	}
	return nil
}

// Build the kustomization
func (k *Kustomize) build() (err error) {
	fs := filesys.MakeFsOnDisk()

	kustomizeBuildMutex.Lock()
	defer kustomizeBuildMutex.Unlock()

	// Kustomize tends to panic in unpredicted ways due to (accidental)
	// invalid object data; recover when this happens to ensure continuity of
	// operations
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
