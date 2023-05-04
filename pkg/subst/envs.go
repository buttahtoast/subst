package subst

import (
	"os"
	"regexp"
	"strings"
)

func GetVariables(regex string) (envs map[string]interface{}, err error) {
	envs = make(map[string]interface{})
	var r *regexp.Regexp

	if regex != "" {
		r, err = regexp.Compile(regex)
		if err != nil {
			return nil, err
		}
	}

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		key := pair[0]
		value := pair[1]

		if value != "" {
			// Verify if regexp matches (Skip no matches)
			if regex != "" {
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
