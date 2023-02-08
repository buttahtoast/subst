package subst

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Shopify/ejson"
	"github.com/buttahtoast/subst/pkg/utils"
	"github.com/geofffranks/spruce"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	publicKeyField = "_public_key"
)

func (b *Build) runEjson() error {
	if !b.cfg.SkipDecrypt {
		err := b.loadEjsonKeys()
		if err != nil {
			return err
		}
	}
	err := b.walkpaths(b.ejsonWalk)
	if err != nil {
		return err
	}

	return nil
}

// Evaluates given EJSON Keys
func (b *Build) loadEjsonKeys() error {
	// try to use k8s secrets
	secretName := b.cfg.EjsonSecret
	if secretName != "" {

		// Set the namespace and secret name
		namespace := b.cfg.EjsonSecretNamespace

		// Get the secret
		secret, err := b.kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
		//if err != nil && !k8serrors.IsNotFound(err) {
		//	return err
		//}
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

func (b *Build) ejsonWalk(path string, info fs.FileInfo, err error) error {
	var (
		data      []byte
		outBuffer bytes.Buffer
	)
	if filepath.Ext(path) == b.cfg.EjsonFilePattern {
		decrypted := false

		// No decryption requested, just read the file
		if !b.cfg.SkipDecrypt {
			// try to decrypt the file
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			for key := range b.keys {
				err = ejson.Decrypt(file, &outBuffer, "", b.keys[key])
				if err != nil {
					continue
				} else {
					data = outBuffer.Bytes()
					decrypted = true
					break
				}
			}

			if b.cfg.MustDecrypt && !decrypted {
				return fmt.Errorf("%s: Could not decrypt with given keys", path)
			}
		}

		// Load file without decryption
		if !decrypted {
			data, err = ioutil.ReadFile(path)
			if err != nil {
				return err
			}
		}

		// Extract data key from ejson file
		d, err := utils.ParseYAML(data)
		if err != nil {
			return err
		}
		// Remove Public Key information
		delete(d, publicKeyField)

		b.Substitutions.Subst, err = spruce.Merge(b.Substitutions.Subst, d)
		if err != nil {
			fmt.Println("t1: %s\n", (err))
			return err
		}
	}

	return err
}
