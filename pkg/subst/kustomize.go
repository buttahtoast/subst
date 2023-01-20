package subst

import (
	"fmt"
	"os"
	"sync"

	"github.com/buttahtoast/subst/pkg/utils"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

var kustomizeBuildMutex sync.Mutex

// Resolves all paths from the kustomization file
func (b *Build) kustomizePaths(path string) error {
	path = utils.ConvertPath(path)
	kz, err := utils.ReadKustomize(path)
	if err != nil {
		return err
	}
	for _, resource := range kz.Resources {
		p := fmt.Sprintf("%v%v", path, resource)
		file, err := os.Stat(p)
		if os.IsNotExist(err) {
			return err
		}
		if file.IsDir() {
			p = utils.ConvertPath(p)
			b.Paths = append(b.Paths, p)
			err := b.kustomizePaths(p)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Build) kustomizeBuild() (build resmap.ResMap, err error) {
	// temporary workaround for concurrent map read and map write bug
	// https://github.com/kubernetes-sigs/kustomize/issues/3659
	kustomizeBuildMutex.Lock()
	defer kustomizeBuildMutex.Unlock()

	fs := filesys.MakeFsOnDisk()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from kustomize build panic: %v", r)
		}
	}()

	buildOptions := &krusty.Options{
		LoadRestrictions: kustypes.LoadRestrictionsNone,
		PluginConfig:     kustypes.DisabledPluginConfig(),
	}

	k := krusty.MakeKustomizer(buildOptions)
	return k.Run(fs, b.root)
}
