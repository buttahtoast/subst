package subst

import (
	"fmt"

	"github.com/buttahtoast/subst/internal/kustomize"
	"github.com/buttahtoast/subst/internal/utils"
	"github.com/buttahtoast/subst/pkg/config"
	"github.com/buttahtoast/subst/pkg/decryptor"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Build struct {
	Manifests     []map[interface{}]interface{}
	Kustomization *kustomize.Kustomize
	Substitutions *Substitutions
	cfg           config.Configuration
	decryptors    []decryptor.Decryptor
	kubeClient    *kubernetes.Clientset
}

func New(config config.Configuration) (build *Build, err error) {

	k, err := kustomize.NewKustomize(config.RootDirectory)
	if err != nil {
		return nil, err
	}

	init := &Build{
		cfg:           config,
		Kustomization: k,
	}

	err = init.initialize()
	if err != nil {
		return nil, err
	}
	SubstitutionsConfig := SubstitutionsConfig{
		EnvironmentRegex: init.cfg.EnvRegex,
		SubstFileRegex:   init.cfg.FileRegex,
	}

	s, err := NewSubstitutions(SubstitutionsConfig, init.decryptors)
	if err != nil {
		return nil, err
	}
	init.Substitutions = s

	err = init.loadSubstitutions()
	if err != nil {
		return nil, err
	}

	return init, err
}

func (b *Build) Build() (err error) {

	// Run Build
	logrus.Debug("substitute manifests")
	for _, manifest := range b.Kustomization.Build.Resources() {
		var c map[interface{}]interface{}

		mBytes, _ := manifest.MarshalJSON()
		for _, d := range b.decryptors {
			if d.IsEncrypted(mBytes) {
				c, err = d.Read(mBytes)
				if err != nil {
					return fmt.Errorf("failed to decrypt")
				}
				continue
			}
		}

		if c == nil {
			m, _ := manifest.AsYAML()

			c, err = utils.ParseYAML(m)
			if err != nil {
				return fmt.Errorf("UnmarshalJSON: %w", err)
			}
		}

		f, err := b.Substitutions.Eval(c, nil, false)
		if err != nil {
			return fmt.Errorf("spruce evaluation failed %s/%s: %s", manifest.GetNamespace(), manifest.GetName(), err)
		}
		b.Manifests = append(b.Manifests, f)
	}
	logrus.Infof("build finished")

	return nil
}

// builds the substitutions interface
func (b *Build) loadSubstitutions() (err error) {

	// Read Substition Files
	err = b.Kustomization.Walk(b.Substitutions.Walk)
	if err != nil {
		return err
	}
	logrus.Debug("subtitution files loaded")

	// Final attempt to evaluate
	eval, err := b.Substitutions.Eval(b.Substitutions.Subst, nil, false)
	if err != nil {
		return err
	}
	b.Substitutions.Subst = eval

	if len(b.Substitutions.Subst) > 0 {
		logrus.Debug("loaded substitutions: ", b.Substitutions.Subst)
	} else {
		logrus.Debug("no substitutions found")
	}

	return nil
}

// initialize decryption
func (b *Build) initialize() (err error) {

	c := decryptor.DecryptorConfig{
		SkipDecrypt: b.cfg.SkipDecrypt,
		MustDecrypt: b.cfg.MustDecrypt,
	}

	ed := decryptor.NewEJSONDecryptor(c, b.cfg.EjsonKey...)
	if err != nil {
		return err
	}
	b.decryptors = append(b.decryptors, ed)

	// SOPS

	if !b.cfg.SkipDecrypt && (b.cfg.SecretName != "" && b.cfg.SecretNamespace != "") {

		var host string
		if b.cfg.KubeAPI != "" {
			host = b.cfg.KubeAPI
		}
		cfg, err := clientcmd.BuildConfigFromFlags(host, b.cfg.Kubeconfig)
		if err == nil {
			b.kubeClient, err = kubernetes.NewForConfig(cfg)
			if err != nil {
				logrus.Warnf("could not load kubernetes client: %s", err)
			} else {
				for _, decr := range b.decryptors {
					err = decr.FromSecret(b.cfg.SecretName, b.cfg.SecretNamespace, b.kubeClient)
					if err != nil {
						logrus.Warnf("failed to load secrets from Kubernetes: %s", err)
					}
				}

			}
		}
	}

	return nil
}
