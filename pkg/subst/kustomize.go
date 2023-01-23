package subst

import (
	"fmt"
	"os"

	"github.com/buttahtoast/subst/pkg/utils"
	generator "github.com/fluxcd/pkg/kustomize"
	"sigs.k8s.io/kustomize/api/resmap"
)

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

	// Create tmp dir.
	tmpDir, err := utils.MkdirTempAbs("", "subst-")
	if err != nil {
		err = fmt.Errorf("tmp dir error: %w", err)
		return nil, err
	}

	 := generator.MakeFsOnDisk()

	// Remove Build directory
	defer os.RemoveAll(tmpDir)

	return generator.Build(tmpDir, b.root)
}
