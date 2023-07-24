package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/ryanchapman/config-container/sidecar"
	_ "github.com/ryanchapman/config-container/sidecar/log"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s '<string_to_decrypt>' '<private_json_keyset>'\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Decrypt a string using a private key (using Google Tink's \"exchange\" method)\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "examples:\n")
	fmt.Fprintf(os.Stderr, "%s 'AQBdqk3v3MinOx72ZUTmRLsMn3KlmU2UUmy+eyzrR03Y4397IExQbNyGzisR0uOoS87CNcH9A1pwfmf/R2fVpQ==' '{\"primaryKeyId\":6138445,\"key\":[{\"keyData\":{\"typeUrl\":\"type.googleapis.com/google.crypto.tink.HpkePrivateKey\",\"value\":\"EioSBggBEAEYAhogk72Nj25pUhzByXgS4piNFoIkixUp0agLpqR1hM/VXEoaIF/bNmedQsiXENLP2shPjEutFFHYtKY1v1CvxrifPpK7\",\"keyMaterialType\":\"ASYMMETRIC_PRIVATE\"},\"status\":\"ENABLED\",\"keyId\":6138445,\"outputPrefixType\":\"TINK\"}]}'\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "would output the decrypted string:\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Hello World\n")
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 {
		usage()
	}

	strToDecrypt := os.Args[1]
	privateJsonKeyset := os.Args[2]
	e := sidecar.NewDecryptor(privateJsonKeyset)

	bytesToDecrypt, err := base64.StdEncoding.DecodeString(strToDecrypt)
	if err != nil {
		log.Fatal().Msgf("could not base64 decode '%s': %s", strToDecrypt, err)
	}

	encryptionContext := []byte(nil)
	plaintext, err := e.Decrypt(bytesToDecrypt, encryptionContext)
	if err != nil {
		log.Panic().Msgf("could not decrypt: %s", err)
	}
	fmt.Printf("%s", plaintext)
}
