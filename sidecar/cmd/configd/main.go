// config-container
// configd/main.go
//
// Store config and encrypted secrets in an config-container (runs as sidecar container).
// This is intended to be super simple, so just one .go file
//
// Goals:
// 1. Store config in git, including encrypted secrets.
//
//  2. Config versioned with code.
//     We want a way to "couple" our config to a particular version of code.  This provides
//     a way to require a new config in code and populate the config values that will be
//     needed by that new code.  For example, if I am currently running v1 of code in production,
//     and I want to add a new config named DB_PORT which specifies the port of the database
//     server, I want to be able to write new code to require DB_PORT as of v2.  I'll also
//     populate DB_PORT for dev, staging and production in the config.yml.tmpl file:
//
//     production:
//     - DB_PORT: "3306"
//     staging:
//     - DB_PORT: "3306"
//     dev:
//     - DB_PORT: "83306"
//
//     (for local dev, assume we use a different port)
//
//  3. Limit who can see production secrets
//     We want to provide a way to store encrypted secrets in git.  But we don't
//     necessarily want all developers to be able to see the decrypted secrets
//     in git.  There may be a workflow where a security engineer can see production
//     secrets, but not normal developers that work in the git repo regularly.
//     If we have three environments (dev, staging, production), maybe the developers
//     can see dev and staging, but not production.  Or maybe they can only see dev.
//
//  4. Only dependency is containers.
//     No config "service" needed
//     (config database like consul, object store like S3 or GCS, secrets service like Vault or AWS Secrets Manager)
//
// We have two containers in play here:
// +----------------+               +------------------+
// | code container |               | config container |
// +----------------+               +------------------+
//
// code container is the normal container you would deploy that has _your_ code in it,
// for example, your API code that will listen on a port for customer traffic.
//
// config container is a new sidecar container that is responsible for providing
// configs (including decrypted secrets) at runtime.
//
// Build time:
//
// In your Dockerfile, you build your code container like you normally would.
// You add a new config container that looks something like this:
//
// FROM config-container AS api-config
// ADD config.yml.tmpl /
//
// You build both code and config containers and tag them with the same version.
//
// Just to provide an example of what a config.yml.tmpl file might look like, consider this:
//
// dev:
//   - DB_PORT: "83306"
//   - DB_USERNAME: "db"
//   - DB_PASSWORD: "db"
//
// staging:
//   - DB_PORT: "3306"
//   - DB_USERNAME: "base64encrypted:abcXYZ="
//   - DB_PASSWORD: "base64encrypted:xyzABC="
//
// production:
//   - DB_PORT: "3306"
//   - DB_USERNAME: "base64encrypted:123abc="
//   - DB_PASSWORD: "base64encrypted:789XYZ="
//
// Here, we have unencrypted configs for the dev environment.  Staging has two configs which are
// encrypted using asymmetric encryption.  Same with production, but the encrypted values are
// different.  The asymmetric keypair used for to encrypt staging configs is different from
// the keypair used for production.
//
// Run time:
//
// We tell our container runtime system (Kubernetes, AWS ECS, etc.) to run the same tag official
// both our code container and our config container.  We also set a dependency that the config
// container has to be healthy before the code container can start.
//
// The config container needs two environment variables to function:
//
//	ENVIRONMENT_NAME, which maps to "dev", "staging" or "production" in above examples
//	PRIVATE_KEY, which is the private side of the keypair used to encrypt the secrets above.
//
// The config container will then load the file "/config.yml.tmpl" and decrypt the secrets using
// PRIVATE_KEY.  If it can't find an environment defined in config.yml.tmpl named ENVIRONMENT_NAME
// or if the decrypted fails, the config container will exit, which should prevent the code
// container from coming up.  Assuming we were trying to upgrade from v1 to v2, this should
// cause the deploy to fail in Kubernetes/ECS/etc. and a rollback to v1 should occur.
//
// Once the config container is up, the code container comes up.  As the entrypoint, the
// code container contacts the config container via http to get the configs needed.  The code
// container loads them into the bash environment, then starts the app.
package main

import (
	//	"bytes"
	"encoding/base64"
	//	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	//	"path"
	"strings"
	//	"text/template"
	//	"gopkg.in/yaml.v3"

	"github.com/ryanchapman/config-container/sidecar"
)

type PublicJSONKeyset []byte
type PrivateJSONKeyset []byte

type AsymmetricKeypair struct {
	PublicJSONKeyset
	PrivateJSONKeyset
}

type AsymmetricKeys map[string]AsymmetricKeypair

// stores kv pairs that will be injected as environment variables into code container
type EnvironmentConfig map[string]string

type Config struct {
	EnvironmentName string
	// this is stored in a Secrets Management System, like AWS Secrets Manger or Vault
	PrivateKey PrivateJSONKeyset
	// these are the decrypted environment variables that we'll provide to the code container
	DecryptedEnvironmentConfig EnvironmentConfig
}

func NewConfig() *Config {
	return &Config{}
}

func Usage() {
	fmt.Printf("\n")
	fmt.Printf("usage: PRIVATE_KEY=\"base64:x\" ENVIRONMENT_NAME=\"x\" %s\n", os.Args[0])
	fmt.Printf("\n")
	fmt.Printf("  PRIVATE_KEY       private key, base64 encoded. Used to decrypt secrets.\n")
	fmt.Printf("  ENVIRONMENT_NAME  environment we are operating in, defined in config.yml.tmpl file.\n")
	fmt.Printf("                    e.g. production, staging, dev\n")
	fmt.Printf("\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  Assuming a private key of \"secret\", running in staging:\n")
	fmt.Printf("  PRIVATE_KEY=\"base64:c2VjcmV0\" ENVIRONMENT_NAME=\"staging\" %s\n", os.Args[0])
}

func (c *Config) parseEnvVars() {
	c.parsePrivateKeyEnvVar()
	c.parseEnvironmentNameEnvVar()
}

func (c *Config) parsePrivateKeyEnvVar() {
	privateKeyFromEnv := os.Getenv("PRIVATE_KEY")
	if privateKeyFromEnv == "" {
		fmt.Printf("FATAL: Env var PRIVATE_KEY must be set\n")
		Usage()
		os.Exit(1)
	}

	after, found := strings.CutPrefix(privateKeyFromEnv, "base64:")
	if !found {
		fmt.Printf("FATAL: Env var PRIVATE_KEY only supports \"base64:\" encoding, got PRIVATE_KEY=%s\n", privateKeyFromEnv)
		os.Exit(1)
	}

	var err error
	c.PrivateKey, err = base64.StdEncoding.DecodeString(after)
	if err != nil {
		fmt.Printf("FATAL: Could not base64 decode data in PRIVATE_KEY, err=%+v\n", err)
		os.Exit(1)
	}
}

func (c *Config) parseEnvironmentNameEnvVar() {
	environmentName := os.Getenv("ENVIRONMENT_NAME")
	if environmentName == "" {
		fmt.Printf("FATAL: Env var ENVIRONMENT_NAME must be set\n")
		os.Exit(1)
	}
	c.EnvironmentName = environmentName
}

type PublicKeyDefinitions map[string]string

/*
func (c *Config) parseTemplate(configFileLocation string) []byte {
	funcs := template.FuncMap{
		"b64decode": c.b64decode,
		"decrypt":   c.decrypt,
		"cfn":       c.get_cfn_output,
	}

	configTmpl, err := template.New(path.Base(configFileLocation)).Funcs(funcs).ParseFiles(configFileLocation)
	if err != nil {
		log.Fatalf("Error parsing template file %s: err=%s", configFileLocation, err)
	}

	buf := new(bytes.Buffer)
	err = configTmpl.Execute(buf, "")
	if err != nil {
		log.Fatalf("Error executing template file %s: err=%s", configFileLocation, err)
	}

	bytes, err := io.ReadAll(buf)
	if err != nil {
		log.Fatalf("Error reading bytes from template buffer, err=%s", err)
	}
	return bytes
}
*/

/*
// environmentName here is "production", "staging", etc.
// corresponds to the top level key in /config.yml.tmpl
// returns the decrypted config for the specified environmentName
func (c *Config) parseConfigFile(environmentName string) {
	// config file location defaults to "/config.yml.tmpl"
	// it can be overridden (for testing) by specifying the full path
	// in the env var CONFIG_FILE_LOCATION
	// e.g. CONFIG_FILE_LOCATION="./config.yml.tmpl"
	configFileLocation := os.Getenv("CONFIG_FILE_LOCATION")
	if configFileLocation == "" {
		configFileLocation = "/config.yml.tmpl"
	}
	_, err := os.Stat(configFileLocation)
	if err != nil {
		fmt.Printf("FATAL: Could not find config file at %s: err=%+v\n", configFileLocation, err)
		os.Exit(1)
	}

	configFileBytes := c.parseTemplate(configFileLocation)

	fmt.Printf("c=%+v", string(configFileBytes))
	configFile := ConfigFile{}
	err = yaml.Unmarshal(configFileBytes, &configFile)
	if err != nil {
		fmt.Printf("FATAL: Could not unmarshal yaml at %s: err=%+v\n", configFileLocation, err)
		os.Exit(1)
	}

	// check if the environment named "production", "staging", etc. exists
	// in the config file at /config.yml
	ourEnv, found := configFile.Environments[c.EnvironmentName]
	if !found {
		fmt.Printf("FATAL: Could not find an environment named \"%s\" in %s\n", c.EnvironmentName, configFileLocation)
		os.Exit(1)
	}

	// TODO(rchapman): implement actual encryption.  Using base64 for proof-of-concept.
	ourEnvDecrypted := EnvironmentConfig{}
	for _, item := range ourEnv {
		for k, v := range item {
			after, found := strings.CutPrefix(v, "base64encrypted:")
			if found {
				decrypted := after
				ourEnvDecrypted[k] = decrypted
			} else {
				ourEnvDecrypted[k] = v
			}
		}
	}

	c.DecryptedEnvironmentConfig = ourEnvDecrypted
}
*/

func getHealthz(w http.ResponseWriter, r *http.Request) {
	logHealthz := os.Getenv("LOG_HEALTHZ")
	if logHealthz != "" {
		log.Printf("got /healthz request, returning HTTP 200 Healthy\n")
	}
	io.WriteString(w, "Healthy\n")
}

func main() {
	// config file location defaults to "/config.yml.tmpl"
	// it can be overridden (for testing) by specifying the full path
	// in the env var CONFIG_FILE_PATH
	// e.g. CONFIG_FILE_PATH="./config.yml.tmpl"
	configFilePath := os.Getenv("CONFIG_FILE_PATH")
	if configFilePath == "" {
		configFilePath = "/config.yml.tmpl"
	}

	templateFuncs := sidecar.NewTemplateFuncs()
	ct := sidecar.NewConfigTemplateFromFile(configFilePath, templateFuncs.FuncMap())

	fmt.Printf("ct=%+v\n", *ct)
	os.Exit(1)

	/*
	   c.ParseConfigs()

	   mux := http.NewServeMux()

	   	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	   		for k, v := range c.DecryptedEnvironmentConfig {
	   			// intentionally made to be compatible with bash style
	   			// env vars
	   			io.WriteString(w, fmt.Sprintf("%s=\"%s\"\n", k, v))
	   		}
	   	})

	   mux.HandleFunc("/healthz", getHealthz)

	   listenPort := os.Getenv("LISTEN_PORT")

	   	if listenPort == "" {
	   		listenPort = "5309"
	   	}

	   log.Printf("About to listen for http traffic on port %s", listenPort)
	   err := http.ListenAndServe(fmt.Sprintf(":%s", listenPort), mux)

	   	if errors.Is(err, http.ErrServerClosed) {
	   		log.Printf("http server shut down normally", err)
	   	} else if err != nil {

	   		log.Printf("error starting http server: %s", err)
	   		os.Exit(1)
	   	}
	*/
}
