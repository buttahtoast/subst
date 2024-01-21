package subst

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"text/template"

	decrypt "github.com/buttahtoast/pkg/decryptors"
	"github.com/buttahtoast/subst/internal/utils"
	"github.com/buttahtoast/subst/internal/wrapper"
	"github.com/geofffranks/spruce"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/kustomize/api/resmap"
)

const (
	specialCharsRegex = "[$&+,:;=?@#|'<>.^*()%!-/]"
)

var (
	matchingRegex *regexp.Regexp
)

type Substitutions struct {
	Subst      map[interface{}]interface{} `yaml:"subst"`
	Config     SubstitutionsConfig         `yaml:"config"`
	decryptors []decrypt.Decryptor
	funcmap    template.FuncMap
	Resources  resmap.ResMap
}

type SubstitutionsConfig struct {
	SubstKey         string `yaml:"subst_key"`
	EnvironmentRegex string `yaml:"environment_regex"`
	SubstFileRegex   string `yaml:"subst_file_pattern"`
	FlattenLowerCase bool   `yaml:"lowercase"`
}

func NewSubstitutions(cfg SubstitutionsConfig, decrypts []decrypt.Decryptor, res resmap.ResMap) (s *Substitutions, err error) {

	if cfg.SubstKey == "" {
		cfg.SubstKey = "subst"
	}

	init := &Substitutions{
		Subst:      make(map[interface{}]interface{}),
		Config:     cfg,
		decryptors: decrypts,
		Resources:  res,
	}

	if init.Config.SubstFileRegex != "" {
		matchingRegex, err = regexp.Compile(init.Config.SubstFileRegex)
		if err != nil {
			return nil, err
		}
		logrus.Debug("using regex: ", init.Config.SubstFileRegex)

	}

	// Load sprig functionMap
	init.funcmap = utils.SprigFuncMap()

	envs, err := GetVariables(cfg.EnvironmentRegex)
	if err != nil {
		return nil, err
	}
	init.Add(utils.ToInterface(envs), true)
	if err != nil {
		return nil, err
	}

	return init, nil
}

// Get returns the Substitutions as map[interface{}]interface{}
func (s *Substitutions) Get() map[interface{}]interface{} {
	return s.Subst
}

// ToMap returns the Substitutions as map[string]interface{}
func (s *Substitutions) GetMap() map[string]interface{} {
	return utils.ToMap(s.Subst)
}

// adds new data to the Substitutions
func (s *Substitutions) Add(data map[interface{}]interface{}, optimistic bool) (err error) {

	tree, err := s.Eval(data, nil, optimistic)
	if err != nil {
		return fmt.Errorf("failed to build subtitutions: %s", err)
	}

	t := s.Get()
	if err != nil {
		return err
	}

	merge, err := spruce.Merge(t, tree)
	if err != nil {
		return fmt.Errorf("could not merge manifest with subtitutions: %s", err)
	}

	s.Subst = merge
	return nil
}

// Merge merges the Substitutions with the given data
func (s *Substitutions) Eval(data map[interface{}]interface{}, substs map[interface{}]interface{}, optimistic bool) (eval map[interface{}]interface{}, err error) {
	if substs == nil {
		substs = s.Get()
		if err != nil {
			return nil, err
		}
	}

	sub := map[interface{}]interface{}{
		s.Config.SubstKey: substs,
	}

	merge, err := spruce.Merge(data, sub)
	if err != nil {
		return nil, fmt.Errorf("could not merge manifest with subtitutions: %s", err)
	}

	if optimistic {
		eval, err = wrapper.SpruceOptimisticEval(merge, []string{s.Config.SubstKey})
		if err != nil {
			return nil, err
		}
	} else {
		tree, err := wrapper.SpruceEval(merge, []string{s.Config.SubstKey})
		if err != nil {
			return nil, err
		}
		eval = tree.Tree
	}

	return eval, nil
}

func (s *Substitutions) Walk(path string, f fs.FileInfo) error {

	if f.IsDir() {
		return nil
	}
	full := filepath.Join(path, f.Name())

	if matchingRegex.MatchString(f.Name()) {
		var c map[interface{}]interface{}
		logrus.Debug("loading: ", full, "")
		file, err := utils.NewFile(full)
		if err != nil {
			return err
		}

		// Read encrypted file
		for _, d := range s.decryptors {
			isEncrypted, err := d.IsEncrypted(file.Byte())
			if err != nil {
				// Handle the error, e.g., log it and continue to the next decryptor
				logrus.Errorf("Error checking encryption for %s: %s", full, err)
				continue
			}
			if isEncrypted {
				logrus.Debugf("decrypted: %s", full)
				file.Byte()
				dm, err := d.Decrypt(file.Byte())
				if err != nil {
					return fmt.Errorf("failed to decrypt %s: %s", full, err)
				}
				c = utils.ToInterface(dm)
				break
			}
		}

		c, err = file.SPRUCE()
		if err != nil {
			// If SPRUCE fails, apply templating and then reattempt SPRUCE
			if tmplErr := file.Template(s.funcmap, s.Subst); tmplErr != nil {
				return fmt.Errorf("failed to template %s: %s", full, tmplErr)
			}

			c, err = file.SPRUCE() // Reattempt SPRUCE after templating
			if err != nil {
				return fmt.Errorf("failed to parse %s: %s", full, err)
			}
		}

		if c[resourcesField] != nil {
			logrus.Debugf("detected resources in %s", full)
			err = s.addResources(c[resourcesField].([]interface{}))
			if err != nil {
				return fmt.Errorf("failed to add resources from %s: %s", full, err)
			}
			delete(c, resourcesField)
		} else {
			err = s.Add(c, true)
			if err != nil {
				return fmt.Errorf("failed to merge %s: %s", full, err)
			}
		}

		logrus.Debug("final: ", full, "")
	}
	return nil
}
