package subst

import (
	"io/fs"
	"path/filepath"

	"github.com/buttahtoast/subst/pkg/utils"
	"github.com/geofffranks/spruce"
)

func (b *Build) varsWalk(path string, info fs.FileInfo, err error) error {
	if filepath.Ext(path) == b.cfg.VarFilePattern {
		file, err := utils.NewFile(path)
		if err != nil {
			return err
		}
		b.Substitutions.Subst.Vars, err = spruce.Merge(b.Substitutions.Subst.Vars, file.Map())
		if err != nil {
			return err
		}
	}
	return err
}
