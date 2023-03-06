package subst

import (
	"fmt"

	"github.com/buttahtoast/subst/internal/kustomize"
	"github.com/buttahtoast/subst/internal/utils"
	"github.com/buttahtoast/subst/internal/wrapper"
	"github.com/buttahtoast/subst/pkg/config"
	"github.com/buttahtoast/subst/pkg/decryptor"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kustomize/api/resmap"
)

type Build struct {
	Manifests     []map[interface{}]interface{}
	kustomization *kustomize.Kustomize
	Substitutions *Substitutions
	cfg           config.Configuration
	kubeClient    *kubernetes.Clientset
}

func New(config config.Configuration) (build *Build, err error) {

	s, err := NewSubstitutions()
	if err != nil {
		return nil, err
	}

	k, err := kustomize.NewKustomize(config.RootDirectory)
	if err != nil {
		return nil, err
	}

	init := &Build{
		cfg:           config,
		Substitutions: s,
		kustomization: k,
	}

	// Load Decryption Keys
	if !init.cfg.SkipDecrypt {
		err = init.loadEjsonKeys()
		if err != nil {
			return nil, err
		}
	}

	init

	return init, err
}

func (b *Build) Build() error {
	var manifests resmap.ResMap

	manifests, err := b.kustomizeBuild()
	if err != nil {
		return err
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

		f, err := b.Substitutions.Eval(d, nil)
		if err != nil {
			return fmt.Errorf("spruce evaluation failed %s/%s: %s", manifest.GetNamespace(), manifest.GetName(), err)
		}

		// Run Environment substitution
		if len(flatEnv) > 0 {
			f, err = b.envsubst(flatEnv, f)
			if err != nil {
				return fmt.Errorf("envsubst failed %s/%s: %s", manifest.GetNamespace(), manifest.GetName(), err)
			}
		}

		b.Manifests = append(b.Manifests, f)
	}
	logrus.Infof("build finished")

	return nil
}

func (b *Build) initializeKubeconfig() (err error) {
	var host string
	if b.cfg.KubeAPI != "" {
		host = b.cfg.KubeAPI
	}
	cfg, err := clientcmd.BuildConfigFromFlags(host, b.cfg.Kubeconfig)
	if err == nil {
		b.kubeClient, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			return err
		}
	}
	return nil
}

// initialize decryption
func (b *Build) initializeDecryptors() (d []decryptor.Decryptor, err error) {

	e := decryptor.NewEJSONDecryptor()

	if err != nil {
		return nil, err
	}

	if b.cfg.SkipDecrypt {
		e.KeysFromSecret()

	}

}

// builds the substitutions interface
func (b *Build) loadSubstitutions() (err error) {

	// Read Substition Files
	err = b.kustomization.Walk(b.Substitutions.Walk)
	if err != nil {
		return err
	}

	// Final attempt to evaluate
	tree, err := wrapper.SpruceEval(b.Substitutions.Subst, []string{})
	if err != nil {
		return err
	}
	b.Substitutions.Subst = tree.Tree
	logrus.Debug("Loaded Substitutions: ", b.Substitutions.Subst)

	return nil
}
