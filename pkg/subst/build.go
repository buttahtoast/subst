package subst

import (
	"path/filepath"

	"github.com/buttahtoast/subst/pkg/config"
	"github.com/geofffranks/spruce"
	"gopkg.in/yaml.v2"
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

type Substitutions struct {
	Subst  map[interface{}]interface{} `yaml:"subst"`
	merger spruce.Merger
}

func (s *Substitutions) tointerface() (map[interface{}]interface{}, error) {
	tmp := make(map[interface{}]interface{})
	yml, err := yaml.Marshal(s)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yml, tmp)
	return tmp, err
}

func (s *Substitutions) merge(doc map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	root, err := s.tointerface()
	t := root["subst"].(map[interface{}]interface{})
	if err != nil {
		return nil, err
	}
	s.merger.Merge(t, doc)
	return t, nil
}

func New(config config.Configuration) (build *Build, err error) {
	result := &Build{
		root: config.RootDirectory,
		cfg:  config,
		keys: config.EjsonKey,
	}

	// Init Substitutions
	result.Substitutions = Substitutions{
		Subst:  make(map[interface{}]interface{}),
		merger: spruce.Merger{AppendByDefault: true},
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

	// Gather all available substitutions
	err = result.gatherSubstituions()
	if err != nil {
		return nil, err
	}

	return result, err
}

func (b *Build) gatherSubstituions() (err error) {

	// Read Vars Files
	err = b.walkpaths(b.varsWalk)
	if err != nil {
		return err
	}

	// Gather Environment
	err = b.readEnvironment()
	if err != nil {
		return err
	}

	// Gather EJSON
	err = b.runEjson()
	if err != nil {
		return err
	}

	return nil
}

func (b *Build) Build() error {
	var manifests resmap.ResMap

	manifests, err := b.kustomizeBuild()
	if err != nil {
		return err
	}

	// Load Substitutions
	substs, err := b.Substitutions.tointerface()
	if err != nil {
		return err
	}

	// Run Spruce
	for _, raw := range manifests.Resources() {

		// Environment Substitution
		manifest, err := b.envsubst(raw)
		if err != nil {
			return err
		}

		// Load Single Manifest
		m, _ := manifest.AsYAML()
		var str map[interface{}]interface{}
		err = yaml.Unmarshal(m, &str)
		if err != nil {
			return err
		}

		substManifest, err := spruce.Merge(str, substs)
		if err != nil {
			return err
		}

		// Evaluate with Spruce
		evaluator := &spruce.Evaluator{
			Tree:     substManifest,
			SkipEval: b.cfg.SkipEvaluation,
		}

		// Evaluate Tree and prune subst
		err = evaluator.Run([]string{"subst"}, nil)
		if err != nil {
			return err
		}
		b.Manifests = append(b.Manifests, evaluator.Tree)
	}

	return nil
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
