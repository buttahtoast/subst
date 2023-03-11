package decryptor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/user"
	"path"
	"sync"

	"github.com/sirupsen/logrus"
	"go.mozilla.org/sops/keyservice"
	"go.mozilla.org/sops/v3"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type SOPSDecryptor struct {
	keys  openpgp.EntityList 
	groups sops.KeyGroup
	// keyServices are the SOPS keyservice.KeyServiceClient's available to the
	// decryptor.
	keyServices      []keyservice.KeyServiceClient
	localServiceOnce sync.Once
}

// Initialize a new SOPS Decryptor
func NewSOPSDecryptor(config DecryptorConfig, keys ...string) *SOPSDecryptor {
	init := &SOPSDecryptor{
		keys: []string{},
	}

	// Static Keys

	return init
}

// File is sops encrypted
func (d *SOPSDecryptor) IsEncrypted(data map[interface{}]interface{}) bool {
	f := data["sops"]
	if f == nil || f == "" {
		return false
	}
	return true
}

//func (d *SOPSDecryptor) Read(data []byte) (content map[interface{}]interface{}, err error) {
// 	store := common.StoreForFormat(formats.FormatFromString("yaml"))
// 	tree, err := store.LoadEncryptedFile(data)
// 	// if err != nil {
// 	// 	return content, err
// 	// }
// 	// tree
// 	// dec := decrypt.Data(data, "yaml")
// 	// // try with each key in array
// 	// for _, key := range d.keys {
// 	// 	tree.Decrypt([]byte(key))
//
// //

// return nil, err// fromSecret
func (d *SOPSDecryptor) fromSecret(secret string, namespace string, client *kubernetes.Clientset) (err error) {
	kubernetesSecret, err := client.CoreV1().Secrets(namespace).Get(context.TODO(), secret, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		logrus.Debug("missing secret: %s/%s", namespace, secret)
	} else if err != nil {
		return err
	}

	// Exract all keys from secret
	for s := range kubernetesSecret.Data {
		key := string(kubernetesSecret.Data[s])
		reader := bytes.NewReader(kubernetesSecret.Data[s])
		block, err := armor.Decode(reader)
		if err != nil {
			return err
		}
		entityReader := packet.NewReader(block.Body)
		entity, err := openpgp.ReadEntity(entityReader)
		if err != nil {
			panic(err)
		}
		d.keys = append(d.keys, entity)


	return err
}

// keyServiceServer returns the SOPS (local) key service clients used to serve
// decryption requests. loadKeyServiceServers() is only configured on the first
// call.
// loadKeyServiceServers loads the SOPS (local) key service clients used to
// func (d *SOPSDecryptor) keyServiceServer() []keyservice.KeyServiceClient {
// 	d.localServiceOnce.Do(func() {
// 		d.loadKeyServiceServers()
// 	})
// 	return d.keyServices
// }

// // loadKeyServiceServers loads the SOPS (local) key service clients used to
// // serve decryption requests for the current set of Decryptor
// // credentials.
// func (d *SOPSDecryptor) loadKeyServiceServers() {
// 	serverOpts := []intkeyservice.ServerOption{
// 		intkeyservice.WithGnuPGHome(d.gnuPGHome),
// 		intkeyservice.WithVaultToken(d.vaultToken),
// 		intkeyservice.WithAgeIdentities(d.ageIdentities),
// 		intkeyservice.WithGCPCredsJSON(d.gcpCredsJSON),
// 	}
// 	if d.azureToken != nil {
// 		serverOpts = append(serverOpts, intkeyservice.WithAzureToken{Token: d.azureToken})
// 	}
// 	serverOpts = append(serverOpts, intkeyservice.WithAWSKeys{CredsProvider: d.awsCredsProvider})
// 	server := intkeyservice.NewServer(serverOpts...)
// 	d.keyServices = append(make([]keyservice.KeyServiceClient, 0), keyservice.NewCustomLocalClient(server))
// }

func gpgHome() string {
	dir := os.Getenv("GNUPGHOME")
	if dir == "" {
		usr, err := user.Current()
		if err != nil {
			return path.Join(os.Getenv("HOME"), "/.gnupg")
		}
		return path.Join(usr.HomeDir, ".gnupg")
	}
	return dir
}

func secRing() string {
	return fmt.Sprint(gpgHome() + "/secring.gpg")
}
