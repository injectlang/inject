// config-container
// injectd/main.go

package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/injectlang/injector"
	_ "github.com/injectlang/injector/log"
	"github.com/rs/zerolog/log"
)

func usage() {
	fmt.Printf("\n")
	fmt.Printf("usage: PRIVATE_JSON_KEYSET_<KEYPAIRNAME>=\"x\" CONTEXT_NAME=\"x\" %s\n", os.Args[0])
	fmt.Printf("\n")
	fmt.Printf("  PRIVATE_JSON_KEYSET_*  private key, base64 encoded. Used to decrypt secrets.\n")
	fmt.Printf("  CONTEXT_NAME           context we are operating in, defined in config.inj.hcl file.\n")
	fmt.Printf("                         e.g. production, staging, dev\n")
	fmt.Printf("\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  Assuming a public/private key named \"DEV202305\" of \"secret\", running in staging:\n")
	fmt.Printf("  PRIVATE_JSON_KEYSET_DEV202305=\"c2VjcmV0\" CONTEXT_NAME=\"staging\" %s\n", os.Args[0])
}

func getHealthz(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msgf("got /healthz request, returning HTTP 200 Healthy\n")
	io.WriteString(w, "Healthy\n")
}

func main() {
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

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Available endpoints:\n")
		io.WriteString(w, "/sh - print exports in a format that can be consumed from a shell\n")
		io.WriteString(w, "      e.g. eval $(curl localhost:5309/sh)\n")
	})

	mux.HandleFunc("/sh", func(w http.ResponseWriter, r *http.Request) {
		for k, v := range context.Exports {
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

	log.Info().Msgf("listening for http traffic on port %s", listenPort)
	err := http.ListenAndServe(fmt.Sprintf(":%s", listenPort), mux)

	if errors.Is(err, http.ErrServerClosed) {
		log.Info().Msgf("http server shut down normally")
	} else if err != nil {
		log.Fatal().Msgf("error starting http server: %s", err)
	}

}
