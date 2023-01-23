package subst

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/buttahtoast/subst/pkg/utils"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	kustypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// buildMutex protects against kustomize concurrent map read/write panic
var kustomizeBuildMutex sync.Mutex

// ReadKustomize reads a kustomization file from a path
func kustomizeFile(path string) (types.Kustomization, error) {
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

// Resolves all paths from the kustomization file
func (b *Build) kustomizePaths(path string) error {
	path = utils.ConvertPath(path)
	kz, err := kustomizeFile(path)
	if err != nil {
		return err
	}

	// Add Patch Locations to paths
	for _, patch := range kz.Patches {
		b.Paths = append(b.Paths, fmt.Sprintf("%v%v", path, filepath.Dir(patch.Path)))
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

	// Create tmp dir.
	//tmpDir, err := utils.MkdirTempAbs("", "subst-")
	//if err != nil {
	//	err = fmt.Errorf("tmp dir error: %w", err)
	//	return nil, err
	//}
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
		PluginConfig:     kustypes.DisabledPluginConfig(),
	}

	k := krusty.MakeKustomizer(buildOptions)
	return k.Run(fs, b.root)
}
