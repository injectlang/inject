//
// entrypoint.go - entrypoint program used in code container to
//                 download secrets from config container and
//                 inject into the environment just before
//                 your code container program (e.g. your API server)
//                 starts.
//

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func usage() {
	fmt.Printf("usage: cmd [args]\n")
	fmt.Printf("\n")
	fmt.Printf("Did you forget to set CMD correctly in your Dockerfile?")
}

func main() {
	logDebug := os.Getenv("LOG_DEBUG")

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	configContainerURL := os.Getenv("CONFIG_CONTAINER_URL")
	if configContainerURL == "" {
		configContainerURL = "http://localhost:5309/sh"
	}

	resp, err := http.Get(configContainerURL)
	if logDebug != "" {
		log.Printf("GET %s returned \"%+v\"\n", configContainerURL, resp)
	}
	if err != nil {
		log.Fatalf("Error calling GET %s, err=%s\n", configContainerURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("GET %s, could not read response body, resp=%v, err=%s\n", configContainerURL, resp, err)
	}

	respBodyLines := string(respBody)
	for _, line := range strings.Split(respBodyLines, "\n") {
		if line == "" || !strings.Contains(line, "=") {
			continue
		}
		// example: line="DB_PORT=\"3306\""
		kvPair := strings.Split(line, "=")
		if len(kvPair) > 2 {
			log.Fatalf("while parsing line \"%s\" and splitting on \"=\", got more than 2 elements", line)
		}
		k := kvPair[0]
		v := kvPair[1]
		// example: k="\"3306\""
		// remove quotes, if any
		v = strings.ReplaceAll(v, "\"", "")
		if logDebug != "" {
			log.Printf("calling os.Setenv(\"%s\",\"%s\")\n", k, v)
		}
		os.Setenv(k, v)
	}

	binary, err := exec.LookPath(os.Args[1])
	if err != nil {
		log.Fatalf("could not lookup full path for %s, err=%v", os.Args[1], err)
	}

	env := os.Environ()
	err = syscall.Exec(binary, os.Args[1:], env)
	if err != nil {
		log.Fatalf("could not fork/exec %s %s, err=%v", binary, os.Args[1:], err)
	}
}
