package decryptor

type DecryptorConfig struct {
	// Decryption is skipped
	SkipDecrypt bool
	// Fails if fail could not be decrypt
	MustDecrypt bool
}

type Decryptor interface {
	IsEncrypted(data []byte) bool
	Read(data []byte) (content map[interface{}]interface{}, err error)
}
