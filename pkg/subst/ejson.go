package subst

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Shopify/ejson"
	"github.com/geofffranks/spruce"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	publicKeyField = "_public_key"
)

type ejsonFile struct {
	Data      map[interface{}]interface{} `json:"-"`
	PublicKey string                      `json:"_public_key"`
}

func (b *Build) runEjson() error {
	if !b.cfg.SkipDecryption {
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
		// Create a new rest config
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return err
		}

		// Create a new clientset
		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return err
		}

		// Set the namespace and secret name
		namespace := b.cfg.EjsonSecretNamespace

		// Get the secret
		secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// add all keys to
		for s := range secret.Data {
			decodedData, err := base64.StdEncoding.DecodeString(string(secret.Data[s]))
			if err != nil {
				return err
			}
			b.keys = append(b.keys, string(decodedData))
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

		// No decryption requested, just read the file
		if b.cfg.SkipDecryption {
			data, err = ioutil.ReadFile(path)
			if err != nil {
				return err
			}
		} else {
			// try to decrypt the file
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			for key := range b.keys {
				err = ejson.Decrypt(file, &outBuffer, "", b.keys[key])
				if err != nil {
					// if decryption fail try other key
					// maybe optimize this by analysing priv key
					continue
				} else {
					// Redirect enrypted data
					data = outBuffer.Bytes()
					// leave loop when decryption retrun zero error
					break
				}
			}
		}

		// Extract data key from ejson file
		i := make(map[string]interface{})
		err := json.Unmarshal([]byte(data), &i)
		if err != nil {
			return err
		}
		// Remove Public Key information
		delete(i, publicKeyField)

		c := convert(i)

		b.Substitutions.Subst, err = spruce.Merge(b.Substitutions.Subst, c)
		if err != nil {
			return err
		}
	}

	return err
}

func convert(inputMap map[string]interface{}) map[interface{}]interface{} {
	var convertedMap = make(map[interface{}]interface{})
	for key, value := range inputMap {
		convertedMap[key] = value
	}
	return convertedMap
}

func extract(jsonStr string, key string) (map[interface{}]interface{}, error) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return nil, err
	}

	result := make(map[interface{}]interface{})
	value, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("key '%s' not found in JSON data", key)
	}

	result[key] = value
	return result, nil
}
