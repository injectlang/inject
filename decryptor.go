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
	"encoding/base64"

	"github.com/google/tink/go/hybrid"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
	"github.com/rs/zerolog/log"
)

type Decryptor struct {
	privateJSONKeyset string
}

func NewDecryptor(privateJSONKeyset string) *Decryptor {
	return &Decryptor{
		privateJSONKeyset: privateJSONKeyset,
	}
}

// Decrypt a byte buffer which was encrypted using Tink's "exchange" method [1].
//
// encryptionContext:
// "In addition to plaintext the encryption takes an extra parameter contextInfo,
//
//	which usually is public data implicit from the context, but should be bound
//	to the resulting ciphertext, i.e. the ciphertext allows for checking the
//	integrity of contextInfo (but there are no guarantees wrt. the secrecy
//	or authenticity of contextInfo).
//
//	contextInfo can be empty or null, but to ensure the correct decryption of
//	a ciphertext the same value must be provided for the decryption operation
//	as was used during encryption (HybridEncrypt)." [2]
//
// In our case, encryptionContext is probably nil.
//
// [1] https://developers.google.com/tink/exchange-data
// [2] https://pkg.go.dev/github.com/google/tink/go@v1.7.0/tink#hdr-Security_guarantees
func (d *Decryptor) Decrypt(ciphertext, encryptionContext []byte) ([]byte, error) {
	privateKeysetHandle, err := insecurecleartextkeyset.Read(
		keyset.NewJSONReader(bytes.NewBufferString(d.privateJSONKeyset)))
	if err != nil {
		return nil, err
	}

	// Retrieve the HybridDecrypt primitive from privateKeysetHandle.
	decPrimitive, err := hybrid.NewHybridDecrypt(privateKeysetHandle)
	if err != nil {
		return nil, err
	}

	log.Debug().Msgf("About to try to decrypt using keyset \"%s\" and ciphertext \"%s\"", d.privateJSONKeyset, base64.StdEncoding.EncodeToString(ciphertext))
	decrypted, err := decPrimitive.Decrypt(ciphertext, encryptionContext)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}
