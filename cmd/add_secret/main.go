package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/injectlang/injector/editfile"
	_ "github.com/injectlang/injector/log"
	"github.com/rs/zerolog/log"
)

func usage() {
	_, _ = fmt.Fprintf(os.Stderr, "usage: %s [--overwrite] [context_name] [key_name] [export_name] [secret_string]\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	_, _ = fmt.Fprintf(os.Stderr, "Add a secret to config.inj.hcl\n")
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	_, _ = fmt.Fprintf(os.Stderr, "To use this program, you must have previously imported a public key using the add_pubkey\n")
	_, _ = fmt.Fprintf(os.Stderr, "program.\n")
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	_, _ = fmt.Fprintf(os.Stderr, "If you omit any parameters, the program will go into interactive mode and ask you to input\n")
	_, _ = fmt.Fprintf(os.Stderr, "all needed parameters.\n")
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	_, _ = fmt.Fprintf(os.Stderr, "example:\n")
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	_, _ = fmt.Fprintf(os.Stderr, "%s staging PROD2022 DB_PASSWORD 'Secret1234'\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	_, _ = fmt.Fprintf(os.Stderr, "would encrypt 'Secret1234' using the public key defined in config.inj.hcl named\n")
	_, _ = fmt.Fprintf(os.Stderr, "PROD2022.  The result would be written to your config.inj.hcl file as:\n")
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	_, _ = fmt.Fprintf(os.Stderr, "context \"staging\" {\n")
	_, _ = fmt.Fprintf(os.Stderr, "    exports = {\n")
	_, _ = fmt.Fprintf(os.Stderr, "        DB_PASSWORD = decrypt(\"PROD2022\", \"AQBdqk2PhwzDSfJoyIIrwPMYV4DLzeUFgKz8+bfp1hxO5Oo1eHaZy7MX0gzxIaqHGopjEn7Gbz4Zt+Q5LER1\")\n")
	_, _ = fmt.Fprintf(os.Stderr, "    }\n")
	_, _ = fmt.Fprintf(os.Stderr, "}\n")
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	_, _ = fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func parseArgs(overwrite *bool, contextName, pubkeyName, exportName, secretStr *string) {
	flag.BoolVar(overwrite, "overwrite", false, "overwrite export if found in config.inj.hcl")
	flag.Usage = usage
	flag.Parse()

	// context name
	if flag.NArg() >= 1 {
		arg := flag.Arg(0)
		*contextName = arg
	}
	// key_name
	if flag.NArg() >= 2 {
		arg := flag.Arg(1)
		*pubkeyName = arg
	}
	// export_name
	if flag.NArg() >= 3 {
		arg := flag.Arg(2)
		*exportName = arg
	}
	// secret_string
	if flag.NArg() >= 4 {
		arg := flag.Arg(3)
		*secretStr = arg
	}
}

func prompt(response *string, promptMsg string) {
	for response == nil || *response == "" {
		fmt.Printf("%s: ", promptMsg)
		_, err := fmt.Scanf("%s", response)
		if err == nil {
			break
		}
	}
}

func main() {
	var overwrite bool
	var contextName string
	var pubkeyName string
	var exportName string
	var secretStr string
	parseArgs(&overwrite, &contextName, &pubkeyName, &exportName, &secretStr)

	configFilePath := os.Getenv("CONFIG_FILE_PATH")
	if configFilePath == "" {
		configFilePath = "config.inj.hcl"
	}
	e := editfile.NewEditConfigFile(configFilePath)

	fmt.Printf("contextName=%s\n", contextName)
	if contextName == "" {
		fmt.Printf("\n")
		contextNames, diags := e.ContextNames()
		if !diags.HasErrors() {
			fmt.Printf("Available context names: %+v\n", contextNames)
		}
		prompt(&contextName, "Context name")
	} else {
		fmt.Printf("using context_name = %s\n", contextName)
	}

	if pubkeyName == "" {
		fmt.Printf("\n")
		pubkeyNames, diags := e.PublicKeyNames()
		if !diags.HasErrors() {
			fmt.Printf("Available public_key names: %+v\n", pubkeyNames)
		}
		prompt(&pubkeyName, "Public key name")
	} else {
		fmt.Printf("using public_key = %s\n", pubkeyName)
	}

	if exportName == "" {
		fmt.Printf("\n")
		exportNames, diags := e.ExportNames()
		if !diags.HasErrors() && overwrite {
			fmt.Printf("Existing export names that can be overwritten: %+v\n", exportNames)
		}
		prompt(&exportName, "Export name")
	} else {
		fmt.Printf("using export_name = %s\n", exportName)
	}

	if secretStr == "" {
		prompt(&secretStr, "Secret string (export value that will be encrypted)")
	} else {
		fmt.Printf("using secret_str = %s\n", secretStr)
	}

	diags := e.AddSecret(contextName, exportName, secretStr, pubkeyName, overwrite)
	if diags.HasErrors() {
		for _, d := range diags {
			fmt.Printf("- %+v\n", d)
		}
		log.Fatal().Msgf("could not add encrypted secret to export \"%s\" in context \"%s\"", exportName, contextName)
	}
}
