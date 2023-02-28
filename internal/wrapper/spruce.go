package wrapper

import (
	"fmt"

	"github.com/geofffranks/spruce"
)

// Run Spruce Eval and return evaluator
func SpruceEval(data map[interface{}]interface{}, prune []string) (eval *spruce.Evaluator, err error) {
	evaluator := &spruce.Evaluator{
		Tree:     data,
		SkipEval: false,
	}

	err = evaluator.Run(prune, nil)
	if err != nil {
		return nil, err
	}

	return evaluator, nil
}

// Trys with eval, if fails, try without eval and trys to return the data tree
func SpruceOptimisticEval(data map[interface{}]interface{}, prune []string) (tree map[interface{}]interface{}, err error) {
	evaluator, err := SpruceEval(data, prune)
	if err != nil {
		// attempt without Evaluation
		evaluator.SkipEval = true
		err = evaluator.Run(prune, nil)
		if err != nil {
			return nil, fmt.Errorf("optimistic eval failed: %s", err)
		}
	}

	return evaluator.Tree, nil
}
