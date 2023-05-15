package subst

import (
	"fmt"
	"log"

	"github.com/buttahtoast/subst/internal/utils"
	"sigs.k8s.io/kustomize/api/provider"
)

const (
	resourcesField = "resources"
)

var (
	defaultResourceFactor = provider.NewDefaultDepProvider().GetResourceFactory()
)

// Add single resource to the Substitution
func (s *Substitutions) addResource(in map[interface{}]interface{}) (err error) {
	// Create the resource
	fmt.Printf("resource: %v\n", in)
	res := defaultResourceFactor.FromMap(utils.ToMap(in))
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}

	err = s.Resources.Append(res)
	if err != nil {
		return err
	}
	return nil
}

// Adds multiple resources to the Substitution
func (s *Substitutions) addResources(resources []interface{}) (err error) {
	for _, v := range resources {
		fmt.Printf("resource: %v\n", v)
		err = s.addResource(v.(map[interface{}]interface{}))
		if err != nil {
			return err
		}
	}
	return nil
}
