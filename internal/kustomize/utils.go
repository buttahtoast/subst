package kustomize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/types"
)

func convertPath(path string) string {
	if path[len(path)-1:] != "/" {
		path = fmt.Sprintf("%v/", path)
	}
	return path
}

// securePaths returns the absolute and relative paths for the provided path,
// guaranteed to be scoped inside the provided root.
// When the given path is absolute, the root is stripped before secure joining
// it on root.
func securePaths(root, path string) (string, string, error) {
	if filepath.IsAbs(path) {
		path = stripRoot(root, path)
	}
	secureAbsPath, err := securejoin.SecureJoin(root, path)
	if err != nil {
		return "", "", err
	}
	return secureAbsPath, stripRoot(root, secureAbsPath), nil
}

func stripRoot(root, path string) string {
	sepStr := string(filepath.Separator)
	root, path = filepath.Clean(sepStr+root), filepath.Clean(sepStr+path)
	switch {
	case path == root:
		path = sepStr
	case root == sepStr:
		// noop
	case strings.HasPrefix(path, root+sepStr):
		path = strings.TrimPrefix(path, root+sepStr)
	}
	return filepath.Clean(filepath.Join("."+sepStr, path))
}

// MkdirTempAbs creates a tmp dir and returns the absolute path to the dir.
// This is required since certain OSes like MacOS create temporary files in
// e.g. `/private/var`, to which `/var` is a symlink.
func mkdirTempAbs(dir, pattern string) (string, error) {
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
func kustomizeFile(path string) (types.Kustomization, error) {
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
