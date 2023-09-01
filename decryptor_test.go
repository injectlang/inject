package injector

import (
	"encoding/base64"
	"fmt"
	"testing"
)

func TestDecryptor(t *testing.T) {
	tests := []struct {
		privateJsonKeyset string
		ciphertextBase64  string
		want              string
	}{
		{
			`{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePrivateKey","value":"EioSBggBEAEYAhogk72Nj25pUhzByXgS4piNFoIkixUp0agLpqR1hM/VXEoaIF/bNmedQsiXENLP2shPjEutFFHYtKY1v1CvxrifPpK7","keyMaterialType":"ASYMMETRIC_PRIVATE"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}
`,
			`AQBdqk1eMvpiEDDnTz9aznyi05u8kzpOAPuUjndbvYIHrKAxNNncQsBjQaUOk06CIUFy+OiJChgSrwGaDLhGOg==`,
			`Hello World`,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			decryptor := NewDecryptor(test.privateJsonKeyset)
			ciphertext, err := base64.StdEncoding.DecodeString(test.ciphertextBase64)
			if err != nil {
				t.Fatalf("could not base64decode ciphertext \"%s\": %s", test.ciphertextBase64, err)
			}
			plaintextBytes, err := decryptor.Decrypt(ciphertext, nil)
			got := string(plaintextBytes)

			if got != test.want {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}
