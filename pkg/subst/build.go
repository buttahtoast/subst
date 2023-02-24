package subst

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/buttahtoast/subst/pkg/config"
	"github.com/buttahtoast/subst/pkg/utils"
	"github.com/geofffranks/spruce"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kustomize/api/resmap"
)

type Build struct {
	root          string
	Paths         []string
	Manifests     []map[interface{}]interface{}
	Substitutions Substitutions
	keys          []string
	cfg           config.Configuration
	// kubeClient a client that knows how to consume kubernetes API.
	kubeClient *kubernetes.Clientset
}

func New(config config.Configuration) (build *Build, err error) {
	result := &Build{
		root: config.RootDirectory,
		cfg:  config,
		keys: config.EjsonKey,
	}

	// Init Substitutions
	result.Substitutions = Substitutions{
		Subst: make(map[interface{}]interface{}),
	}

	// Load Kubernetes Client
	var host string
	if config.KubeAPI != "" {
		host = config.KubeAPI
	}
	cfg, err := clientcmd.BuildConfigFromFlags(host, result.cfg.Kubeconfig)
	if err == nil {
		result.kubeClient, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, err
		}
	}

	// Gather all releveant paths
	err = result.kustomizePaths(result.root)
	if err != nil {
		return nil, err
	}

	// Load Decryption Keys
	if !result.cfg.SkipDecrypt {
		err = result.loadEjsonKeys()
		if err != nil {
			return nil, err
		}
	}

	// Gather Environment
	err = result.readEnvironment()
	if err != nil {
		return nil, err
	}
	logrus.Debug("Loaded Environment Variables")

	// Read Substition Files
	err = result.walkpaths(result.substFiles)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Loaded Substitutions: ", result.Substitutions.Subst)

	return result, err
}

func (b *Build) Build() error {
	var manifests resmap.ResMap

	manifests, err := b.kustomizeBuild()
	if err != nil {
		return err
	}

	// Load Substitutions
	substs, err := b.Substitutions.Tointerface()
	if err != nil {
		return fmt.Errorf("could not convert substitutions: %w", err)
	}

	// Flattened Environment Variables
	flatEnv, err := b.Substitutions.Flatten()
	if err != nil {
		return fmt.Errorf("failed to flatten environment: %w", err)
	}

	// Run Build
	logrus.Debug("substitute manifests")
	for _, manifest := range manifests.Resources() {

		// Load Single Manifest
		m, _ := manifest.AsYAML()

		d, err := utils.ParseYAML(m)
		if err != nil {
			return fmt.Errorf("UnmarshalJSON: %w", err)
		}

		substManifest, err := spruce.Merge(d, substs)
		if err != nil {
			return fmt.Errorf("could not merge manifest with subtitutions: %s", err)
		}

		// Evaluate with Spruce
		evaluator := &spruce.Evaluator{
			Tree:     substManifest,
			SkipEval: b.cfg.SkipEvaluation,
		}

		// Evaluate Tree and prune subst
		err = evaluator.Run([]string{"subst"}, nil)
		if err != nil {
			return fmt.Errorf("spruce evaluation failed: %s", err)
		}

		f := evaluator.Tree
		// Run Environment substitution
		if len(flatEnv) > 0 {
			f, err = b.envsubst(flatEnv, evaluator.Tree)
			if err != nil {
				return fmt.Errorf("environment substitution failed: %s", err)
			}
		}

		b.Manifests = append(b.Manifests, f)
	}
	logrus.Infof("build finished")

	return nil
}

// builds the substitutions interface
func (b *Build) buildSubstitutions() (err error) {

	// Read Substition Files
	err = b.walkpaths(b.substFiles)
	if err != nil {
		return err
	}

	// Final attempt to evaluate
	tree, err := utils.SpruceEval(b.Substitutions.Subst, []string{})
	if err != nil {
		return err
	}
	b.Substitutions.Subst = tree.Tree
	logrus.Debug("Loaded Substitutions: ", b.Substitutions.Subst)

	return nil
}

func (b *Build) substFiles(path string, info fs.FileInfo, err error) error {

	// Load File
	ext := filepath.Ext(path)
	if ext == b.cfg.VarFilePattern || ext == b.cfg.EjsonFilePattern {
		var c map[interface{}]interface{}
		logrus.Debug("Loading file: ", path, "")
		file, err := utils.NewFile(path)
		if err != nil {
			return err
		}

		switch ext {
		case b.cfg.VarFilePattern:
			logrus.Debug("vars file: ", path, "")
			c, err = file.SPRUCE()
			if err != nil {
				return fmt.Errorf("Encountered error %s: %s", path, err)
			}
		case b.cfg.EjsonFilePattern:
			logrus.Debug("ejson file: ", path, "")
			c, err = b.decrypt(*file)
			if err != nil {
				return fmt.Errorf("failed to read %s: %s", path, err)
			}
		}

		// Merge Substitutions
		err = b.Substitutions.Add(c)
		if err != nil {
			return fmt.Errorf("failed to merge %s: %s", path, err)
		}

	}
	return err
}

// Generic funtion to walk all paths and run a function on each file
func (b *Build) walkpaths(fn filepath.WalkFunc) error {
	for path := range b.Paths {
		err := filepath.Walk(b.Paths[path], fn)
		if err != nil {
			return err
		}
	}
	return nil
}
