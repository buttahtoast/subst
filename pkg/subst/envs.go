package subst

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/drone/envsubst"
	"sigs.k8s.io/kustomize/api/resource"
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

// Read Environment Variables for substitution
func (b *Build) readEnvironment() error {
	envs := make(map[string]string)

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		key := pair[0]
		value := pair[1]

		// Verify if regexp matches (Skip no matches)
		if b.cfg.AllowedEnvRegex != "" {
			r, err := regexp.Compile(b.cfg.AllowedEnvRegex)
			if err != nil {
				return err
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
	if envs != nil {
		b.Substitutions.Subst["env"] = envs
	}

	return nil
}

func (b *Build) envsubst(res *resource.Resource) (*resource.Resource, error) {
	if b.Substitutions.Subst["env"] != nil {
		vars := b.Substitutions.Subst["env"].(map[string]string)

		resData, err := res.AsYAML()
		if err != nil {
			return nil, err
		}

		// Check if resource has substition disabled
		if substitionDisabled(res.GetAnnotations()) || substitionDisabled(res.GetLabels()) {
			return nil, nil
		}

		if len(vars) > 0 {
			r, _ := regexp.Compile(varsubRegex)
			for v := range vars {
				if !r.MatchString(v) {
					return nil, fmt.Errorf("'%s' var name is invalid, must match '%s'", v, varsubRegex)
				}
			}
			output, err := envsubst.Eval(string(resData), func(s string) string {
				return vars[s]
			})
			if err != nil {
				return nil, fmt.Errorf("variable substitution failed: %w", err)
			}

			jsonData, err := yaml.YAMLToJSON([]byte(output))
			if err != nil {
				return nil, fmt.Errorf("YAMLToJSON: %w", err)
			}
			err = res.UnmarshalJSON(jsonData)
			if err != nil {
				return nil, fmt.Errorf("UnmarshalJSON: %w", err)
			}
		}
	}

	return res, nil

}

func substitionDisabled(annotations map[string]string) bool {
	disabledValue := "disabled"
	for _, key := range substituteAnnotationKeys {
		if val, _ := annotations[key]; val == disabledValue {
			return true
		}
	}
	return false
}
