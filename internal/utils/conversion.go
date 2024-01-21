package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/geofffranks/simpleyaml"
	"github.com/starkandwayne/goutils/ansi"
	"gopkg.in/yaml.v2"
)

// Convert converts a map[string]interface{} to a map[interface{}]interface{}.
func ToInterface(inputMap map[string]interface{}) map[interface{}]interface{} {
	var convertedMap = make(map[interface{}]interface{})
	for key, value := range inputMap {
		convertedMap[key] = value
	}
	return convertedMap
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

func ConvertPath(path string) string {
	if path[len(path)-1:] != "/" {
		path = fmt.Sprintf("%v/", path)
	}
	return path
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

// create a golang function which prints map[interface{}]interface{} as yaml
func PrintYAML(data map[interface{}]interface{}) error {
	y, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	if _, err := writer.WriteString("---\n"); err != nil {
		return err
	}
	if _, err := writer.Write(y); err != nil {
		return err
	}

	return nil
}

// create a golang function which prints map[interface{}]interface{}
func PrintJSON(data map[interface{}]interface{}) error {
	j, err := json.MarshalIndent(mapify(data), "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(j))
	return err
}
