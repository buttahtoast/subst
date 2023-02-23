package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/geofffranks/simpleyaml"
	"github.com/starkandwayne/goutils/ansi"
	"gopkg.in/yaml.v2"
)

func ConvertPath(path string) string {
	if path[len(path)-1:] != "/" {
		path = fmt.Sprintf("%v/", path)
	}
	return path
}

// convert map[interface{}]interface{} recursive to map[string]string
func ToMap(i map[interface{}]interface{}) map[string]interface{} {
	out := mapify(i)
	return out
}

// convert map[string]interface{} to map[string]string (Recursion)
func mapify(input map[interface{}]interface{}) map[string]interface{} {
	output := make(map[string]interface{})
	for k, v := range input {
		switch vv := v.(type) {
		case map[interface{}]interface{}:
			output[k.(string)] = mapify(vv)
		default:
			output[k.(string)] = vv
		}
	}
	return output
}

// Convert converts a map[string]interface{} to a map[interface{}]interface{}.
func ConvertMap(inputMap map[string]interface{}) map[interface{}]interface{} {
	var convertedMap = make(map[interface{}]interface{})
	for key, value := range inputMap {
		convertedMap[key] = value
	}
	return convertedMap
}

// Convert YAML/JSON data to a map[interface{}]interface{}.
func ParseYAML(data []byte) (map[interface{}]interface{}, error) {
	y, err := simpleyaml.NewYaml(data)
	if err != nil {
		return nil, err
	}

	if empty_y, _ := simpleyaml.NewYaml([]byte{}); *y == *empty_y {
		fmt.Println("YAML doc is empty, creating empty hash/map")
		return make(map[interface{}]interface{}), nil
	}

	doc, err := y.Map()

	if err != nil {
		if _, arrayErr := y.Array(); arrayErr == nil {
			return nil, err
		}
		return nil, ansi.Errorf("@R{Root of YAML document is not a hash/map}: %s\n", err.Error())
	}

	return doc, nil
}

// MkdirTempAbs creates a tmp dir and returns the absolute path to the dir.
// This is required since certain OSes like MacOS create temporary files in
// e.g. `/private/var`, to which `/var` is a symlink.
func MkdirTempAbs(dir, pattern string) (string, error) {
	tmpDir, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		return "", fmt.Errorf("error evaluating symlink: %w", err)
	}
	return tmpDir, nil
}

/* create a golang function which prints map[interface{}]interface{} as yaml */
func PrintYAML(data map[interface{}]interface{}) error {
	y, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Printf("---\n%s", string(y))
	return err
}

/* create a golang function which prints map[interface{}]interface{} as json */
func PrintJSON(data map[interface{}]interface{}) error {
	j, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", string(j))
	return err
}
