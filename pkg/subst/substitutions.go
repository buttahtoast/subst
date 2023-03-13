package subst

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/buttahtoast/subst/internal/utils"
	"github.com/buttahtoast/subst/internal/wrapper"
	"github.com/buttahtoast/subst/pkg/decryptor"
	"github.com/geofffranks/spruce"
	flat "github.com/nqd/flat"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
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
	decryptors []decryptor.Decryptor
}

type SubstitutionsConfig struct {
	SubstKey         string `yaml:"subst_key"`
	EnvironmentRegex string `yaml:"environment_regex"`
	SubstFileRegex   string `yaml:"subst_file_pattern"`
	FlattenLowerCase bool   `yaml:"lowercase"`
}

func NewSubstitutions(cfg SubstitutionsConfig, decrypts []decryptor.Decryptor) (s *Substitutions, err error) {

	if cfg.SubstKey == "" {
		cfg.SubstKey = "subst"
	}

	init := &Substitutions{
		Subst:      make(map[interface{}]interface{}),
		Config:     cfg,
		decryptors: decrypts,
	}

	if init.Config.SubstFileRegex != "" {
		matchingRegex, err = regexp.Compile(init.Config.SubstFileRegex)
		if err != nil {
			return nil, err
		}
		logrus.Debug("using regex: ", init.Config.SubstFileRegex)

	}

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
func (s *Substitutions) Get() (map[interface{}]interface{}, error) {
	output := make(map[interface{}]interface{})
	yml, err := yaml.Marshal(s.Subst)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yml, output)
	return output, err
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

	t, err := s.Get()
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
		substs, err = s.Get()
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

func (s *Substitutions) Flatten() (map[string]string, error) {
	output := make(map[string]string)

	b := s.GetMap()
	f, err := flat.Flatten(b, &flat.Options{Prefix: "", Delimiter: "_", MaxDepth: 50, Safe: false})
	if err != nil {
		return nil, err
	}

	// Convert to map[string]string
	for k, v := range f {
		key := k

		// Convert unallowed characters to underscores
		r := regexp.MustCompile(specialCharsRegex)
		if r.MatchString(key) {
			key = r.ReplaceAllString(key, "_")
		}

		// Convert to lowercase
		if s.Config.FlattenLowerCase {
			key = strings.ToLower(key)
		}
		output[key] = fmt.Sprint(v)
	}

	return output, nil
}

func (s *Substitutions) Walk(path string, info fs.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}

	if matchingRegex.MatchString(info.Name()) {
		var c map[interface{}]interface{}
		logrus.Debug("loading: ", path, "")
		file, err := utils.NewFile(path)
		if err != nil {
			return err
		}

		c, err = file.SPRUCE()
		if err != nil {
			return fmt.Errorf("failed error %s: %s", path, err)
		}

		// Read encrypted file
		for _, d := range s.decryptors {
			if d.IsEncrypted(file.Byte()) {
				logrus.Debugf("decrypted: %s", path)
				c, err = d.Read(file.Byte())
				if err != nil {
					return fmt.Errorf("failed to decrypt %s: %s", path, err)
				}
				continue
			}
		}

		err = s.Add(c, true)
		if err != nil {
			return fmt.Errorf("failed to merge %s: %s", path, err)
		}

	}
	return err
}
