package subst

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/buttahtoast/subst/internal/utils"
	"github.com/buttahtoast/subst/internal/wrapper"
	"github.com/geofffranks/spruce"
	flat "github.com/nqd/flat"
	"gopkg.in/yaml.v2"
)

const (
	specialCharsRegex = "[$&+,:;=?@#|'<>.^*()%!-/]"
)

type Substitutions struct {
	Subst  map[interface{}]interface{} `yaml:"subst"`
	Config SubstitutionsConfig         `yaml:"config"`
}

type SubstitutionsConfig struct {
	SubstKey         string `yaml:"subst_key"`
	FlattenLowerCase bool   `yaml:"lowercase"`
}

func NewSubstitutions() (s *Substitutions, err error) {
	cfg := SubstitutionsConfig{
		SubstKey:         "subst",
		FlattenLowerCase: false,
	}

	init := &Substitutions{
		Subst:  make(map[interface{}]interface{}),
		Config: cfg,
	}

	// Load Environment Variables
	//err = init.getEnv()
	//if err != nil {
	//	return nil, err
	//}

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
func (s *Substitutions) Add(data map[interface{}]interface{}) (err error) {

	tree, err := s.Eval(data, nil)
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
func (s *Substitutions) Eval(data map[interface{}]interface{}, substs map[interface{}]interface{}) (merge map[interface{}]interface{}, err error) {
	if substs == nil {
		substs, err = s.Get()
		if err != nil {
			return nil, err
		}
	}

	sub := map[interface{}]interface{}{
		s.Config.SubstKey: substs,
	}

	merge, err = spruce.Merge(data, sub)
	if err != nil {
		return nil, fmt.Errorf("could not merge manifest with subtitutions: %s", err)
	}

	tree, err := wrapper.SpruceOptimisticEval(merge, []string{s.Config.SubstKey})
	if err != nil {
		return nil, fmt.Errorf("could not merge manifest with subtitutions: %s", err)
	}
	return tree, nil
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
