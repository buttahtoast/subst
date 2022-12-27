package utils

import (
	"fmt"
	"os"

	"github.com/buttahtoast/subst/pkg/config"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	kustypes "sigs.k8s.io/kustomize/api/types"
)

var (
	paths []string
)

// ReadKustomize reads a kustomization file from a path
func readKustomize(path string) (types.Kustomization, error) {
	kz := types.Kustomization{}
	for _, kfilename := range konfig.RecognizedKustomizationFileNames() {
		if _, err := os.Stat(path + kfilename); err == nil {
			kzBytes, err := os.ReadFile(path + kfilename)
			if err != nil {
				println(err)
			}
			err = kz.Unmarshal(kzBytes)

			return kz, err
		}
	}
	return kz, fmt.Errorf("no kustomization file found in %v", path)
}

func GetKustomizePaths(path string) ([]string, error) {
	path = convertPath(path)
	kz, err := readKustomize(path)
	for _, resource := range kz.Resources {
		p := fmt.Sprintf("%v%v", path, resource)
		file, err := os.Stat(p)
		if os.IsNotExist(err) {
			return paths, err
		}
		if file.IsDir() {
			p = convertPath(p)
			paths = append(paths, p)
			GetKustomizePaths(p)
		}
	}
	return paths, err
}

func Build(config config.Configuration) (resmap.ResMap, error) {

	fs := filesys.MakeFsOnDisk()

	buildOptions := &krusty.Options{
		DoLegacyResourceSort: true,
		LoadRestrictions:     kustypes.LoadRestrictionsNone,
		AddManagedbyLabel:    false,
		DoPrune:              false,
		PluginConfig:         kustypes.DisabledPluginConfig(),
	}
	k := krusty.MakeKustomizer(buildOptions)
	m, err := k.Run(fs, config.RootDirectory)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func convertPath(path string) string {
	if path[len(path)-1:] != "/" {
		path = fmt.Sprintf("%v/", path)
	}
	return path
}
