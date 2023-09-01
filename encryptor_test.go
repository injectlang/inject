package injector

import (
	"fmt"
	"testing"
)

func TestEncryptor(t *testing.T) {
	tests := []struct {
		publicJsonKeyset  string
		privateJsonKeyset string
		plaintext         string
		encryptionContext []byte
		want              string
	}{
		{
			`{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePublicKey","value":"EgYIARABGAIaIJO9jY9uaVIcwcl4EuKYjRaCJIsVKdGoC6akdYTP1VxK","keyMaterialType":"ASYMMETRIC_PUBLIC"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}`,
			`{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePrivateKey","value":"EioSBggBEAEYAhogk72Nj25pUhzByXgS4piNFoIkixUp0agLpqR1hM/VXEoaIF/bNmedQsiXENLP2shPjEutFFHYtKY1v1CvxrifPpK7","keyMaterialType":"ASYMMETRIC_PRIVATE"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}`,
			`Hello World`,
			[]byte{},
			`Hello World`,
		},
		{
			`{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePublicKey","value":"EgYIARABGAIaIJO9jY9uaVIcwcl4EuKYjRaCJIsVKdGoC6akdYTP1VxK","keyMaterialType":"ASYMMETRIC_PUBLIC"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}`,
			`{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePrivateKey","value":"EioSBggBEAEYAhogk72Nj25pUhzByXgS4piNFoIkixUp0agLpqR1hM/VXEoaIF/bNmedQsiXENLP2shPjEutFFHYtKY1v1CvxrifPpK7","keyMaterialType":"ASYMMETRIC_PRIVATE"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}`,
			`Hello World`,
			[]byte("aContext"),
			`Hello World`,
		},
	}

	// because the result of encrypting is different every time, we need to go through a full encrypt->decrypt
	// to verify encryption works.
	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			encryptor := NewEncryptor(test.publicJsonKeyset)
			ciphertext, err := encryptor.Encrypt([]byte(test.plaintext), test.encryptionContext)
			if err != nil {
				t.Fatalf("could not encrypt string \"%s\": %s", test.plaintext, err)
			}

			decryptor := NewDecryptor(test.privateJsonKeyset)
			plaintextBytes, err := decryptor.Decrypt(ciphertext, test.encryptionContext)
			got := string(plaintextBytes)

			if got != test.want {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func TestEncryptor_Fails(t *testing.T) {
	tests := []struct {
		description       string
		publicJsonKeyset  string
		privateJsonKeyset string
		plaintext         string
		encryptionContext []byte
		decryptionContext []byte
	}{
		{
			"different encryption context used for encryption vs decryption",
			`{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePublicKey","value":"EgYIARABGAIaIJO9jY9uaVIcwcl4EuKYjRaCJIsVKdGoC6akdYTP1VxK","keyMaterialType":"ASYMMETRIC_PUBLIC"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}`,
			`{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePrivateKey","value":"EioSBggBEAEYAhogk72Nj25pUhzByXgS4piNFoIkixUp0agLpqR1hM/VXEoaIF/bNmedQsiXENLP2shPjEutFFHYtKY1v1CvxrifPpK7","keyMaterialType":"ASYMMETRIC_PRIVATE"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}`,
			`Hello World`,
			[]byte("aContext"),
			[]byte("differentContext"),
		},
		{
			"private key does not match public key",
			`{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePublicKey","value":"EgYIARABGAIaIJO9jY9uaVIcwcl4EuKYjRaCJIsVKdGoC6akdYTP1VxK","keyMaterialType":"ASYMMETRIC_PUBLIC"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}`,
			`{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePrivateKey","value":"EioSBggBEAEYAhogk72Nj25pUhzByXgS4piNFoIkixUp0agLpqR1hM/VXEoaIF/bNmedQsiXENLP2shPjEutFFHYtKY1v1CvxrifPPk9","keyMaterialType":"ASYMMETRIC_PRIVATE"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}`,
			`Hello World`,
			[]byte("aContext"),
			[]byte("aContext"),
		},
	}

	// because the result of encrypting is different every time, we need to go through a full encrypt->decrypt
	// to verify encryption works.
	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d-%s", i, test.description), func(t *testing.T) {
			encryptor := NewEncryptor(test.publicJsonKeyset)
			ciphertext, err := encryptor.Encrypt([]byte(test.plaintext), test.encryptionContext)
			if err != nil {
				t.Fatalf("could not encrypt string \"%s\": %s", test.plaintext, err)
			}

			decryptor := NewDecryptor(test.privateJsonKeyset)
			plaintextBytes, err := decryptor.Decrypt(ciphertext, test.decryptionContext)
			if err == nil {
				got := string(plaintextBytes)
				want := "<DECRYPTION_FAILED>"
				t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, want)
			}
		})
	}
}
