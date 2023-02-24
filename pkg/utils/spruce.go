package utils

import (
	"fmt"

	"github.com/geofffranks/spruce"
)

func SpruceEval(data map[interface{}]interface{}, prune []string) (eval *spruce.Evaluator, err error) {
	evaluator := &spruce.Evaluator{
		Tree: data,
	}

	err = evaluator.Run(prune, nil)
	if err != nil {
		return nil, err
	}

	return evaluator, nil
}

// Trys with eval, if fails, try without eval
func SpruceOptimisticEval(data map[interface{}]interface{}) (tree map[interface{}]interface{}, err error) {
	evaluator := &spruce.Evaluator{
		Tree: data,
	}

	err = evaluator.Run([]string{}, nil)
	if err != nil {
		// attempt without Evaluation
		evaluator.SkipEval = true
		err = evaluator.Run([]string{}, nil)
		if err != nil {
			return nil, fmt.Errorf("eval failed: %s", err)
		}
	}

	return evaluator.Tree, nil
}
