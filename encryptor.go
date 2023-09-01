package injector

// Encryption and decryption using asymmetric encryption
// Based on https://developers.google.com/tink/exchange-data
//
// Our PrivateJSONKeyset is stored in a secret manager like
// like AWS Secrets Manager, GCP Secrets Manager, or Hashicorp Vault.
//
// To generate a keypair, use:
//
// brew tap tink-crypto/tink-tinkey https://github.com/tink-crypto/tink-tinkey
// brew install tinkey
//
// tinkey create-keyset --key-template=DHKEM_X25519_HKDF_SHA256_HKDF_SHA256_AES_256_GCM --out private_keyset.cfg
//
// Note that this keyset has the secret key information in cleartext.
//
// The intent here is to base64 encode the private_keyset.cfg file and
// store in a secret manager.
//
// At runtime, you get this secret from a secret manager and inject into
// the config container as an environment variable.
//
//
// To create the public side, run
//
// tinkey create-public-keyset --in private_keyset.cfg
//
//
import (
	"bytes"

	"github.com/google/tink/go/hybrid"
	"github.com/google/tink/go/keyset"
)

type Encryptor struct {
	PublicJSONKeyset string
}

func NewEncryptor(publicJSONKeyset string) *Encryptor {
	return &Encryptor{
		PublicJSONKeyset: publicJSONKeyset,
	}
}

// Encrypt a byte buffer using Tink's "exchange" method.
//
// plaintext: byte buffer containing plaintext you want to encrypt
// encryptionContext: see Decrypt() for explanation of this.  You can just
// use nil unless you have a specific use case for this.
func (e *Encryptor) Encrypt(plaintext, encryptionContext []byte) ([]byte, error) {
	publicKeysetHandle, err := keyset.ReadWithNoSecrets(
		keyset.NewJSONReader(bytes.NewBufferString(e.PublicJSONKeyset)))
	if err != nil {
		return nil, err
	}

	// Retrieve the HybridEncrypt primitive from publicKeysetHandle.
	encPrimitive, err := hybrid.NewHybridEncrypt(publicKeysetHandle)
	if err != nil {
		return nil, err
	}

	ciphertext, err := encPrimitive.Encrypt(plaintext, encryptionContext)
	if err != nil {
		return nil, err
	}

	return ciphertext, nil
}
