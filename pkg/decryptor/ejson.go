package decryptor

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Shopify/ejson"
	"github.com/buttahtoast/subst/internal/utils"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

const (
	publicKeyField = "_public_key"
)

type EjsonDecryptor struct {
	keys   []string
	Config DecryptorConfig // Embedded
}

// Initialize a new EJSON Decryptor
func NewEJSONDecryptor(config DecryptorConfig, keys ...string) *EjsonDecryptor {
	init := &EjsonDecryptor{
		keys:   []string{},
		Config: config,
	}

	if len(keys) > 0 {
		for _, key := range keys {
			init.AddKey(key)
		}
	}

	return init
}

func (d *EjsonDecryptor) IsEncrypted(data []byte) bool {
	c, err := utils.ParseYAML(data)
	if err != nil {
		return false
	}
	f := c[publicKeyField]
	if f == nil || f == "" {
		return false
	}
	return true
}

func (d *EjsonDecryptor) AddKey(key string) {
	privkeyBytes, err := hex.DecodeString(strings.TrimSpace(key))
	if err != nil {
		return
	}

	if len(privkeyBytes) != 32 {
		logrus.Error("invalid private key", privkeyBytes)
		return
	}

	logrus.Debug("adding ejson key: ", privkeyBytes)

	d.keys = append(d.keys, strings.TrimSpace(key))
}

// Load Keys from Kubernetes Secret
func (d *EjsonDecryptor) FromSecret(secret string, namespace string, client *kubernetes.Clientset) (err error) {
	kubernetesSecret, err := client.CoreV1().Secrets(namespace).Get(context.TODO(), secret, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		logrus.Debug("missing secret: %s/%s", namespace, secret)
	} else if err != nil {
		return err
	}

	// Exract all keys from secret
	for s := range kubernetesSecret.Data {
		key := string(kubernetesSecret.Data[s])
		d.AddKey(key)
	}

	return nil
}

// Read an ejson file
// Skip decryption still removes the publicKeyField
func (d *EjsonDecryptor) Read(data []byte) (content map[interface{}]interface{}, err error) {
	conent, err := d.Decrypt(data)
	if err != nil {
		return nil, err
	}

	// Extract data key from ejson file
	y, err := utils.ParseYAML(conent)
	if err != nil {
		return nil, err
	}
	// Remove Public Key information
	delete(y, publicKeyField)

	return y, err
}

// Attempts to decrypt an ejson file with the given keys
func (d *EjsonDecryptor) Decrypt(data []byte) (content []byte, err error) {
	var outputBuffer bytes.Buffer

	decrypted := false
	f := bytes.NewReader(data)
	if !d.Config.SkipDecrypt {

		// Try all loaded keys
		for key := range d.keys {
			err = ejson.Decrypt(f, &outputBuffer, "", string(d.keys[key]))
			if err != nil {
				continue
			} else {
				decrypted = true
				break
			}
		}

		// Check if file was decrypted (and must be)
		if d.Config.MustDecrypt && !decrypted {
			e := fmt.Errorf("could not decrypt with given keys")
			// This error happens, if the file is not properly encrypted (or not encrypted at all)
			// Considered an error.
			if err != nil && err.Error() == "invalid message format" {
				e = fmt.Errorf("content is not encrypted with ejson")
			}
			return nil, e
		}
	}

	if outputBuffer.Bytes() != nil {
		return outputBuffer.Bytes(), nil
	}

	return data, nil
}
