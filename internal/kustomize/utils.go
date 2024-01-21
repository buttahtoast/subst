package kustomize

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/types"
)

func convertPath(path string) string {
	// Efficiently ensure path ends with a slash
	if len(path) > 0 && path[len(path)-1] != filepath.Separator {
		path += string(filepath.Separator)
	}
	return path
}

func kustomizeFile(path string) (types.Kustomization, error) {
	kz := types.Kustomization{}
	for _, kfilename := range konfig.RecognizedKustomizationFileNames() {
		fullPath := filepath.Join(path, kfilename)
		if _, err := os.Stat(fullPath); err == nil {
			kzBytes, err := os.ReadFile(fullPath)
			if err != nil {
				return kz, err
			}
			err = kz.Unmarshal(kzBytes)

			return kz, err
		}
	}
	return kz, fmt.Errorf("no kustomization file found in %v", path)
}
