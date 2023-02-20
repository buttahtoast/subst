package subst

import (
	flat "github.com/nqd/flat"
)

type Substitutions struct {
	Subst map[interface{}]interface{} `yaml:"subst"`
}

func (s *Substitutions) ToMap() map[string]interface{} {
	out := s.mapify(s.Subst)
	return out
}

func (s *Substitutions) mapify(input map[interface{}]interface{}) map[string]interface{} {
	output := make(map[string]interface{})
	for k, v := range input {
		switch vv := v.(type) {
		case map[interface{}]interface{}:
			output[k.(string)] = s.mapify(vv)
		default:
			output[k.(string)] = vv
		}
	}
	return output
}

func (s *Substitutions) Flatten() (map[string]interface{}, error) {
	b := s.ToMap()
	return (flat.Flatten(b, &flat.Options{Prefix: "HRRLE", Delimiter: "_", MaxDepth: 10, Safe: false}))
}
