package subst

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/buttahtoast/subst/internal/utils"
	"github.com/drone/envsubst"
	jsoniter "github.com/json-iterator/go"
	"sigs.k8s.io/yaml"
)

const (
	// varsubRegex is the regular expression used to validate
	// the var names before substitution
	varsubRegex   = "^[_[:alpha:]][_[:alpha:][:digit:]]*$"
	DisabledValue = "disabled"
)

var (
	substituteAnnotationKeys = []string{"kustomize.toolkit.fluxcd.io/substitute"}
)

func GetVariables(regex string) (envs map[string]interface{}, err error) {
	envs = make(map[string]interface{})

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		key := pair[0]
		value := pair[1]

		if value != "" {
			// Verify if regexp matches (Skip no matches)
			if regex != "" {
				r, err := regexp.Compile(regex)
				if err != nil {
					return nil, err
				}
				if !r.MatchString(key) {
					continue
				}
			}
			// Rewrite ArgoCD Environment Variables
			argoRegex := regexp.MustCompile("^ARGOCD_ENV_")
			if argoRegex.MatchString(key) {
				key = strings.ReplaceAll(key, "ARGOCD_ENV_", "")
			}
			envs[key] = value
		}
	}
	return envs, nil
}

func Envsubst(vars map[string]string, res map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	r, _ := regexp.Compile(varsubRegex)
	for v := range vars {
		if !r.MatchString(v) {
			return nil, fmt.Errorf("'%s' var name is invalid, must match '%s'", v, varsubRegex)
		}
	}

	if res["metadata"] != nil {
		metadata := res["metadata"].(map[interface{}]interface{})
		if metadata["annotations"] != nil {
			annotations := metadata["annotations"].(map[interface{}]interface{})
			if substitionDisabled(annotations) {
				return res, nil
			}
		}
	}

	// jsoniter.Marshal() is used instead of json.Marshal() because it supports nested map types
	z, err := jsoniter.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("error converting manifest: %w", err)
	}
	// Run substitution
	output, err := envsubst.Eval(string(z), func(s string) string {
		return vars[s]
	})

	if err != nil {
		return nil, fmt.Errorf("variable substitution failed: %w", err)
	}

	jsonData, err := yaml.YAMLToJSON([]byte(output))
	if err != nil {
		return nil, fmt.Errorf("YAMLToJSON: %w %s", err, output)
	}

	d, err := utils.ParseYAML(jsonData)
	if err != nil {
		return nil, fmt.Errorf("UnmarshalJSON: %w", err)
	}

	return d, nil
}

func substitionDisabled(annotations map[interface{}]interface{}) bool {
	disabledValue := "disabled"
	for _, key := range substituteAnnotationKeys {
		if val, _ := annotations[key]; val == disabledValue {
			return true
		}
	}
	return false
}
