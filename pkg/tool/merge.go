package tool

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/buttahtoast/subst/pkg/config"
	"github.com/buttahtoast/subst/pkg/utils"
	"github.com/geofffranks/spruce"
	"gopkg.in/yaml.v2"
)

func Merge(config config.Configuration, state Spruce) ([]map[interface{}]interface{}, error) {
	manifests := []map[interface{}]interface{}{}

	// Create Kustomize Build
	kusto, err := utils.Build(config)
	if err != nil {
		return nil, err
	}

	// Merge Substitutions with each Manifest
	for _, manifest := range kusto.Resources() {
		log.Printf("\n \n %v", manifest)
		m, _ := manifest.AsYAML()
		var tmp map[interface{}]interface{}
		log.Printf("HERE 1")
		err := yaml.Unmarshal(m, &tmp)
		if err != nil {
			return nil, err
		}
		log.Printf("HERE 2")
		s, err := state.ToInterface()
		//fmt.Printf("\n \n %v", s)
		if err != nil {
			return nil, err
		}
		merged, err := spruce.Merge(s, tmp)
		log.Printf("HERE 3")
		if err != nil {
			return nil, err
		}
		log.Printf("HERE 4")
		// Evaluate with Spruce
		evaluator := &spruce.Evaluator{
			Tree:     merged,
			SkipEval: false,
		}
		err = evaluator.Run([]string{}, nil)
		fmt.Println(err)
		if err != nil {
			return nil, err
		}
		delete(evaluator.Tree, "subst")
		fmt.Println(evaluator.Tree)
		//fmt.Printf("\n \n\n \n\n \n\n \n%v \n \n", eval)
		//return eval.Tree, nil
		manifests = append(manifests, evaluator.Tree)

		//fmt.Printf(" herehrere %v, %v", eval.Tree, err)
	}

	return manifests, nil

}

func PrettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		fmt.Println(string(b))
	}
	return
}
