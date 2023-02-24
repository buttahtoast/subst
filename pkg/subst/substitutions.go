package subst

import (
	"fmt"
	"regexp"

	"github.com/buttahtoast/subst/pkg/utils"
	flat "github.com/nqd/flat"
	"gopkg.in/yaml.v2"
)

type Substitutions struct {
	Subst map[interface{}]interface{} `yaml:"subst"`
}

//type FlattenConfig struct {
//	LowerCase
//	flat.Options
//	Subst map[interface{}]interface{} `yaml:"subst"`
//}

func (s *Substitutions) Tointerface() (map[interface{}]interface{}, error) {
	tmp := make(map[interface{}]interface{})
	yml, err := yaml.Marshal(s)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yml, tmp)
	return tmp, err
}

func (s *Substitutions) ToMap() map[string]interface{} {
	return utils.ToMap(s.Subst)
}

func (s *Substitutions) Flatten() (map[string]string, error) {
	output := make(map[string]string)

	b := s.ToMap()
	f, err := flat.Flatten(b, &flat.Options{Prefix: "", Delimiter: "_", MaxDepth: 10, Safe: false})
	if err != nil {
		return nil, err
	}

	// Convert to map[string]string
	for k, v := range f {
		key := k

		// Convert unallowed characters to underscores
		r := regexp.MustCompile(utils.SpecialCharsRegex)
		if r.MatchString(key) {
			key = r.ReplaceAllString(key, "_")
		}
		output[key] = fmt.Sprintf("%v", v)
	}

	return output, nil
}
