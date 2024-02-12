package subst

import (
	"context"
	"fmt"

	decrypt "github.com/buttahtoast/pkg/decryptors"
	ejson "github.com/buttahtoast/pkg/decryptors/ejson"
	sops "github.com/buttahtoast/pkg/decryptors/sops"
	"github.com/buttahtoast/subst/internal/kustomize"
	"github.com/buttahtoast/subst/internal/utils"
	"github.com/buttahtoast/subst/pkg/config"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Build struct {
	Manifests     []map[interface{}]interface{}
	Kustomization *kustomize.Kustomize
	Substitutions *Substitutions
	cfg           config.Configuration
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

	return init, err
}

func (b *Build) BuildSubstitutions() (err error) {
	decryptors, cleanups, err := b.decryptors()
	if err != nil {
		return err
	}

	defer func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}()

	SubstitutionsConfig := SubstitutionsConfig{
		EnvironmentRegex: b.cfg.EnvRegex,
		SubstFileRegex:   b.cfg.FileRegex,
	}

	b.Substitutions, err = NewSubstitutions(SubstitutionsConfig, decryptors, b.Kustomization.Build)
	if err != nil {
		return err
	}

	err = b.loadSubstitutions()
	if err != nil {
		return err
	}
	return nil

}

func (b *Build) Build() (err error) {

	if b.Substitutions == nil {
		logrus.Debug("no resources to build")
		return nil
	}

	decryptors, cleanups, err := b.decryptors()
	if err != nil {
		return err
	}

	defer func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}()

	// Run Build
	logrus.Debug("substitute manifests")
	for _, manifest := range b.Substitutions.Resources.Resources() {
		var c map[interface{}]interface{}

		mBytes, _ := manifest.MarshalJSON()
		for _, d := range decryptors {
			isEncrypted, err := d.IsEncrypted(mBytes)
			if err != nil {
				logrus.Errorf("Error checking encryption for %s: %s", mBytes, err)
				continue
			}
			if isEncrypted {
				dm, err := d.Decrypt(mBytes)
				if err != nil {
					return fmt.Errorf("failed to decrypt %s: %s", mBytes, err)
				}
				c = utils.ToInterface(dm)
				break
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

	return nil
}

// builds the substitutions interface
func (b *Build) loadSubstitutions() (err error) {

	// Read Substition Files
	err = b.Kustomization.Walk(b.Substitutions.Walk)
	if err != nil {
		return err
	}

	// Final attempt to evaluate
	eval, err := b.Substitutions.Eval(b.Substitutions.Subst, nil, false)
	if err != nil {
		return fmt.Errorf("spruce evaluation failed")
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
func (b *Build) decryptors() (decryptors []decrypt.Decryptor, cleanups []func(), err error) {

	c := decrypt.DecryptorConfig{
		SkipDecrypt: b.cfg.SkipDecrypt,
	}

	ed, err := ejson.NewEJSONDecryptor(c, "", b.cfg.EjsonKey...)
	if err != nil {
		return nil, nil, err
	}
	decryptors = append(decryptors, ed)

	if b.cfg.SopsTempKeyring {
		sd, sopsCleanup, err := sops.NewSOPSTempDecryptor(c)
		if err != nil {
			return nil, nil, err
		}
		cleanups = append(cleanups, sopsCleanup)
		decryptors = append(decryptors, sd)
	} else {
		decryptors = append(decryptors, sops.NewSOPSDecryptor(c, b.cfg.SopSKeyring))
	}

	if b.cfg.SecretSkip {
		return
	}

	if !b.cfg.SkipDecrypt && (b.cfg.SecretName != "" && b.cfg.SecretNamespace != "") {

		var host string
		if b.cfg.KubeAPI != "" {
			host = b.cfg.KubeAPI
		}
		cfg, err := clientcmd.BuildConfigFromFlags(host, b.cfg.Kubeconfig)
		if err == nil {
			b.kubeClient, err = kubernetes.NewForConfig(cfg)
			if err != nil {
				logrus.Debug("could not load kubernetes client: %s", err)
			} else {
				ctx := context.Background()
				for _, decr := range decryptors {
					err = decr.KeysFromSecret(b.cfg.SecretName, b.cfg.SecretNamespace, b.kubeClient, ctx)
					if err != nil {
						logrus.Debug("failed to load secrets from Kubernetes: %s", err)
					}
				}

			}
		}
	}

	return
}
