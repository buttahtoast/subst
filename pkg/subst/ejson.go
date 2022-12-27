package subst

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/Shopify/ejson"
	"github.com/geofffranks/spruce"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ejsonFile struct {
	Data      map[interface{}]interface{} `json:"-"`
	PublicKey string                      `json:"_public_key"`
}

func (b *Build) loadEjsonKeys() error {

	// try to use k8s secrets
	secretName := b.cfg.EjsonSecret
	if secretName != "" {
		// Create a new rest config
		cfg, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}

		// Create a new clientset
		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			panic(err)
		}

		// Set the namespace and secret name
		namespace := b.cfg.EjsonSecretNamespace

		// Get the secret
		secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil {
			panic(err)
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
	if filepath.Ext(path) == b.cfg.EjsonFilePattern {
		data, err := b.readFile(path)
		if err != nil {
			return err
		}
		tmp, err := ExtractKey(string(data), "data")
		if err != nil {
			return err
		}
		b.Substitutions.Subst.Secrets, err = spruce.Merge(b.Substitutions.Subst.Secrets, tmp)
	}
	return err
}

func (b *Build) readFile(path string) ([]byte, error) {
	var (
		err  error
		data []byte
	)

	// Try encrypt with every key
	for key := range b.keys {
		data, err = ejson.DecryptFile(path, "", b.keys[key])
		if err != nil {
			// if decryption fail try other key
			// maybe optimize this by analysing priv key
			continue
		}
		// leave loop when decryption retrun zero error
		break
	}

	return data, err
}

func ExtractKey(jsonStr string, key string) (map[interface{}]interface{}, error) {
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
