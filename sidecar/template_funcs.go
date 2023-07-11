package sidecar

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"text/template"
)

type TemplateFuncs struct {
	funcMap template.FuncMap
}

func NewTemplateFuncs() *TemplateFuncs {
	tf := TemplateFuncs{}

	funcs := template.FuncMap{
		"b64decode": tf.b64decode,
		"decrypt":   tf.decrypt,
		"cfn":       tf.getCfnOutput,
	}

	tf.funcMap = funcs

	return &tf
}

func (tf *TemplateFuncs) FuncMap() template.FuncMap {
	return tf.funcMap
}

func (tf *TemplateFuncs) b64decode(s string) string {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		log.Panicf("Could not base64 decode string \"%s\"", s)
	}
	return string(decoded)
}

func (tf *TemplateFuncs) decrypt(publicKeyName string, b64ciphertext string) string {
	privateEnvName := fmt.Sprintf("PRIVATE_JSON_KEYSET_%s", publicKeyName)
	privateJSONKeyset := os.Getenv(privateEnvName)
	if privateJSONKeyset == "" {
		log.Panicf("Env var %s must be set", privateEnvName)
	}
	privateJSONKeyset = tf.b64decode(privateJSONKeyset)
	decryptor := NewDecryptor(privateJSONKeyset)

	ciphertext := tf.b64decode(b64ciphertext)
	encryptionContext := []byte(nil)
	plaintext, err := decryptor.Decrypt([]byte(ciphertext), encryptionContext)
	if err != nil {
		log.Panicf("could not decrypt: %+v", err)
	}
	return string(plaintext)
}

func (f *TemplateFuncs) getCfnOutput(region, stackName, outputName string) string {
	// TODO(rchapman): add support for querying CFN outputs
	return "NOT_IMPLEMENTED"
}
