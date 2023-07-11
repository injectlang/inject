package sidecar

// This file handles parsing a yaml config file into a struct
// called ConfigFile.  Template parsing does not happen here.

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type ConfigFile struct {
	Meta         ConfigFileMeta                         `yaml:"_meta"`
	Environments map[string]ConfigFileEnvironmentConfig `yaml:"environments"`
	originalYaml string
}

type ConfigFileMeta struct {
	PublicKeyDefs map[string]string `yaml:"public_keys"`
}

type ConfigFileEnvironmentConfig map[string]string

const DEFAULT_CONFIG_FILE_PATH = "config.yml.tmpl"

func NewConfigFile(path string) *ConfigFile {
	if path == "" {
		path = DEFAULT_CONFIG_FILE_PATH
	}
	yamlBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Panicf("error reading config file at path %s: %+v", path, err)
	}
	return NewConfigFileFromYaml(string(yamlBytes))
}

func NewConfigFileFromYaml(yaml string) *ConfigFile {
	cf := ConfigFile{}
	yamlBytes := []byte(yaml)
	err := cf.parse(yamlBytes)
	if err != nil {
		log.Printf("could not parse this yaml:")
		print(yamlBytes)
		log.Panicf("could not unmarshal yaml: %+v", err)
	}
	return &cf
}

func (cf *ConfigFile) Yaml() string {
	return cf.originalYaml
}

func (cf *ConfigFile) parse(configFileBytes []byte) error {
	err := yaml.Unmarshal(configFileBytes, &cf)
	if err != nil {
		return err
	}
	cf.originalYaml = string(configFileBytes)
	return nil
}

// given a public key in a file like public_keyset.json, base64 encode the contents of the file,
// then add to _meta:public_keys in the ConfigFile
// e.g.
// _meta:
//
//	public_keys:
//	  PROD2022: |
//	    BASE64ENCODINGOFPUBLICKEYSETJSON
func (cf *ConfigFile) AddPubkey(pubkeyName, pathToPublicKeysetJson string) {
	// pubkeyName must start with an uppercase letter
	re := regexp.MustCompile(`^[A-Z]`)
	if !re.MatchString(pubkeyName) {
		log.Panicf("when adding a public key to config file, name of public key must start with an uppercase letter, got %s", pubkeyName)
	}

	// pubkeyName must be all uppercase letters and numbers
	re = regexp.MustCompile(`^[A-Z0-9]+`)
	if !re.MatchString(pubkeyName) {
		log.Panicf("when adding a public key to config file, name of public key must be uppercase letters and numbers, got %s", pubkeyName)
	}

	// read in contents of file pathToPublicKeysetJson
	publicKeysetJson, err := ioutil.ReadFile(pathToPublicKeysetJson)
	if err != nil {
		log.Panicf("could not read file %s: %+v", pathToPublicKeysetJson, err)
	}

	// base64 encode contents of file pathToPublicKeysetJson
	b64PublicJsonKeyset := base64.StdEncoding.EncodeToString(publicKeysetJson)

	cf.Meta.PublicKeyDefs[pubkeyName] = b64PublicJsonKeyset

	// write up to 64 characters of base64 encoded string to yaml key named pubkeyName
}

func print(bytes []byte) {
	lines := strings.Split(string(bytes), "\n")
	for i, line := range lines {
		fmt.Printf("%3.3d  %s\n", i+1, line)
	}
}
