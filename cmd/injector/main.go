// config-container
// injectd/main.go

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/injectlang/injector"
	_ "github.com/injectlang/injector/log"
	"github.com/rs/zerolog/log"
)

func usage() {
	fmt.Printf("\n")
	fmt.Printf("Inject a config file into the environment before executing a program.\n")
	fmt.Printf("\n")
	fmt.Printf("This is typically used as a way to get configs, including decrypted secrets, into\n")
	fmt.Printf("the environment variables of a daemon.\n")
	fmt.Printf("\n")
	fmt.Printf("usage: PRIVATE_JSON_KEYSET_<KEYPAIRNAME>=\"x\" CONTEXT_NAME=\"x\" %s <next_program>\n", os.Args[0])
	fmt.Printf("\n")
	fmt.Printf("Required environment variables:\n")
	fmt.Printf("  PRIVATE_JSON_KEYSET_*  private key, base64 encoded. Used to decrypt secrets.\n")
	fmt.Printf("  CONTEXT_NAME           context we are operating in, defined in config.inj.hcl file.\n")
	fmt.Printf("                         e.g. production, staging, dev\n")
	fmt.Printf("\n")
	fmt.Printf("Required args:\n")
	fmt.Printf("  next_program           program (usually a daemon) to execute next.\n")
	fmt.Printf("\n")
	fmt.Printf("Optional:\n")
	fmt.Printf("  CONFIG_FILE_PATH       path to config.inj.hcl file, defaults to /config.inj.hcl\n")
	fmt.Printf("\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  Assuming a public/private key named \"DEV202305\" of \"secret\", running in staging:\n")
	fmt.Printf("  PRIVATE_JSON_KEYSET_DEV202305=\"c2VjcmV0\" CONTEXT_NAME=\"staging\" %s\n", os.Args[0])
}

func main() {
	if len(os.Args) < 2 {
		usage()
		log.Fatal().Msgf("next_program must be specified")
	}

	contextName := os.Getenv("CONTEXT_NAME")
	if contextName == "" {
		usage()
		log.Fatal().Msgf("Environment variable CONTEXT_NAME must be set.")
	}
	// config file location defaults to "/config.inj.hcl"
	// it can be overridden (for testing) by specifying the full path
	// in the env var CONFIG_FILE_PATH
	// e.g. CONFIG_FILE_PATH="./config.inj.hcl"
	configFilePath := os.Getenv("CONFIG_FILE_PATH")
	if configFilePath == "" {
		configFilePath = "/config.inj.hcl"
	}

	cf := injector.NewConfigFile(configFilePath)

	var context injector.Context
	found := false
	for _, c := range cf.Contexts {
		if c.Name == contextName {
			found = true
			context = c
			break
		}
	}
	if !found {
		log.Fatal().Msgf("No context named \"%s\" defined in \"%s\"", contextName, configFilePath)
	}

	for k, v := range context.Exports {
		log.Debug().Msgf("calling os.Setenv(\"%s\",\"%s\")\n", k, v)
		os.Setenv(k, v)
	}

	binary, err := exec.LookPath(os.Args[1])
	if err != nil {
		log.Fatal().Msgf("could not lookup full path for %s, err=%v", os.Args[1], err)
	}

	// remove config.inj.hcl to prevent it from accidentally being exposed by next process
	err = os.Remove(configFilePath)
	if err != nil {
		log.Warn().Msgf("could not delete config file at %s, err=%v", configFilePath, err)
	} else {
		log.Debug().Msgf("deleted config file at %s", configFilePath)
	}

	// copy our env, then delete private key env vars
	// like PRIVATE_JSON_KEYSET_DEV202305
	env := os.Environ()
	// envvar looks like "SHELL=/bin/bash"
	for i, envvar := range env {
		if strings.HasPrefix(envvar, "PRIVATE_JSON_KEYSET_") {
			envvarName, _, found := strings.Cut(envvar, "=")
			if !found {
				envvarName = envvar
			}
			log.Debug().Msgf("removing var %s from env", envvarName)
			env = append(env[:i], env[i+1:]...)
		}
	}

	err = syscall.Exec(binary, os.Args[1:], env)
	if err != nil {
		log.Fatal().Msgf("could not exec %s %s, err=%v", binary, os.Args[1:], err)
	}
}
