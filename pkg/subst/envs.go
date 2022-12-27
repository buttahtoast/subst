package subst

import (
	"os"
	"regexp"
	"strings"
)

// Read Environment Variables for substitution
func (b *Build) readEnvironment() error {
	b.Substitutions.Subst.Env = make(map[string]string)
	r, _ := regexp.Compile("^ARGOCD_ENV_.*$")
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		key := pair[0]
		value := pair[1]

		// Rewrite ArgoCD Environment Variables
		if r.MatchString(key) {
			key = strings.ReplaceAll(key, "ARGOCD_ENV_", "")
		}
		b.Substitutions.Subst.Env[key] = value
	}
	return nil
}
