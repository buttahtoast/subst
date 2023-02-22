package subst

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Shopify/ejson"
	"github.com/buttahtoast/subst/pkg/utils"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

const (
	publicKeyField = "_public_key"
)

// Evaluates given EJSON Keys
func (b *Build) loadEjsonKeys() error {
	// try to use k8s secrets
	secretName := b.cfg.EjsonSecret
	if secretName != "" {

		// Set the namespace and secret name
		namespace := b.cfg.EjsonSecretNamespace
		klog.SetLogger(logr.Discard())

		// Get the secret
		if b.kubeClient != nil {
			secret, err := b.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
			if k8serrors.IsNotFound(err) {
				logrus.Debug("Could not find secret: %s/%s", namespace, secretName)
			} else if err != nil {
				return err
			}

			// add all keys to
			for s := range secret.Data {
				key := string(secret.Data[s])
				logrus.Debug("Loaded EJSON key: ", key)
				b.keys = append(b.keys, key)
			} // Read from Kubernetes Secret

		}
	}

	if len(b.keys) == 0 {
		logrus.Debug("No EJSON keys loaded")
	} else {
		logrus.Debug("Found EJSON Keys: ", b.keys)
	}

	return nil
}

func (b *Build) decrypt(file utils.File) (d map[interface{}]interface{}, err error) {
	var (
		outBuffer bytes.Buffer
	)
	data := file.Byte()
	decrypted := false
	f := bytes.NewReader(data)

	if !b.cfg.SkipDecrypt {
		for key := range b.keys {
			err = ejson.Decrypt(f, &outBuffer, "", b.keys[key])
			if err != nil {
				logrus.Debug("Attempt tp decrypt failed ", file.Path)
				continue
			} else {
				data = outBuffer.Bytes()
				decrypted = true
				break
			}
		}

		if b.cfg.MustDecrypt && !decrypted {
			e := fmt.Errorf("%s: Could not decrypt with given keys", file.Path)
			// We could not decrypt the file
			if err != nil && err.Error() == "invalid message format" {
				e = fmt.Errorf("%s: File is not encrypted with ejson", file.Path)
			}
			return nil, e
		}

	}

	// Extract data key from ejson file
	y, err := utils.ParseYAML(data)
	if err != nil {
		return nil, err
	}
	// Remove Public Key information
	delete(y, publicKeyField)

	return y, err
}
