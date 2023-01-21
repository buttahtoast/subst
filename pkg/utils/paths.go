package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/types"
)

func ConvertPath(path string) string {
	if path[len(path)-1:] != "/" {
		path = fmt.Sprintf("%v/", path)
	}
	return path
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

// ReadKustomize reads a kustomization file from a path
func ReadKustomize(path string) (types.Kustomization, error) {
	kz := types.Kustomization{}
	for _, kfilename := range konfig.RecognizedKustomizationFileNames() {
		if _, err := os.Stat(path + kfilename); err == nil {
			kzBytes, err := os.ReadFile(path + kfilename)
			if err != nil {
				println(err)
			}
			err = kz.Unmarshal(kzBytes)

			return kz, err
		}
	}
	return kz, fmt.Errorf("no kustomization file found in %v", path)
}
