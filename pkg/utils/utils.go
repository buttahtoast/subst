package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

func ConvertPath(path string) string {
	if path[len(path)-1:] != "/" {
		path = fmt.Sprintf("%v/", path)
	}
	return path
}

// Convert converts a map[string]interface{} to a map[interface{}]interface{}.
func ConvertMap(inputMap map[string]interface{}) map[interface{}]interface{} {
	var convertedMap = make(map[interface{}]interface{})
	for key, value := range inputMap {
		convertedMap[key] = value
	}
	return convertedMap
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
	fmt.Printf("---\n%s\n", string(y))
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
