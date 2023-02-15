package subst

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Shopify/ejson"
	"github.com/buttahtoast/subst/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// k8serrors "k8s.io/apimachinery/pkg/api/errors"

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

		// Get the secret
		secret, err := b.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// add all keys to
		for s := range secret.Data {
			key := string(secret.Data[s])
			b.keys = append(b.keys, key)
		} // Read from Kubernetes Secret
	}
	return nil
}

func (b *Build) decrypt(file utils.File) (err error, d map[interface{}]interface{}) {
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
				continue
			} else {
				data = outBuffer.Bytes()
				decrypted = true
				break
			}
		}

		if b.cfg.MustDecrypt && !decrypted {
			return fmt.Errorf("%s: Could not decrypt with given keys", file.Path), nil
		}

	}

	// Extract data key from ejson file
	y, err := utils.ParseYAML(data)
	if err != nil {
		return err, nil
	}
	// Remove Public Key information
	delete(y, publicKeyField)

	return err, y
}
