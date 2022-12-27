package ejson

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/Shopify/ejson"
	"github.com/buttahtoast/subst/pkg/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Ejson struct {
	config  config.Configuration `json:"-"`
	secrets map[interface{}]interface{}
	keys    []string
}

type Form struct {
	Data      map[interface{}]interface{} `json:"-"`
	PublicKey string                      `json:"_public_key"`
}

func (e *Ejson) GetSecrets() map[interface{}]interface{} {
	return e.secrets
}

func New(config config.Configuration) (Ejson, error) {
	ejs := Ejson{
		config: config,
	}
	return ejs, ejs.loadKeys()
}
func (e *Ejson) loadKeys() error {
	// Read inline keys
	e.keys = e.config.EjsonKey

	// try to use k8s secrets
	if e.config.EjsonSecret != "" {
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
		namespace := e.config.EjsonSecretNamespace
		secretName := e.config.EjsonSecret

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
			e.keys = append(e.keys, string(decodedData))
		} // Read from Kubernetes Secret

	}
	return nil

}
func (e *Ejson) Walk(path string, info fs.FileInfo, err error) error {
	if filepath.Ext(path) == ".ejson" {
		data, err := e.readFile(path)
		if err != nil {
			return err
		}
		tmp, err := ExtractKey(string(data), "data")
		if err != nil {
			return err
		}
		e.secrets = tmp
	}
	return err
}
func (e *Ejson) readFile(path string) ([]byte, error) {
	var (
		err  error
		data []byte
	)

	// Try encrypt with every key
	for key := range e.keys {
		data, err = ejson.DecryptFile(path, "", e.keys[key])
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
