package sidecar

// This file handles parsing a yaml config file into a struct
// called ConfigFile.  Template parsing does not happen here.

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/rs/zerolog/log"
	"github.com/ryanchapman/config-container/sidecar/customfunc"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"os"
)

// There are a few sections in a config.hcl file:
//
// custom_function "greet" {
// }
//
// custom_function "getCfnOutput" {
// }
//
// public_keys {
// }
//
// environment "staging" {
// }
//
// environment "prod" {
// }
//
// We are only concerned with the environment blocks in this struct
// Decoding/encoding of other blocks is done manually in the other
// libraries.
type ConfigFile struct {
	Environments EnvironmentsConfig `hcl:"environment,block"`
	Remain       hcl.Body           `hcl:",remain"`
}

type EnvironmentsConfig []EnvironmentConfig

type EnvironmentConfig struct {
	Name string            `hcl:"name,label"`
	Vars map[string]string `hcl:"vars,optional"`
	// environment variables used by application
	Exports map[string]string `hcl:"exports"`
}

// type PublicKeysConfig []PublicKeyConfig
//
//type PublicKeyConfig struct {
//	Name   string `hcl:"name,label"`
//	Base64 string `hcl:"base64"`
//}

const DEFAULT_CONFIG_FILE_PATH = "config.hcl"

func NewConfigFile(path string) *ConfigFile {
	if path == "" {
		path = DEFAULT_CONFIG_FILE_PATH
	}

	src, err := ioutil.ReadFile(path)
	if err != nil {
		log.Panic().Msgf("error reading config file at path %s: %s", path, err)
	}

	parser := hclparse.NewParser()
	parser.ParseHCL(src, path)

	customFuncs, _, diags := customfunc.DecodeCustomFunctions(parser, src, path, nil)
	allFuncs := NewHclFuncs(customFuncs).FuncMap()
	configFile := ConfigFile{}
	// decode hcl file using custom functions and built-in funcs in hcl_funcs.go
	d := decode(parser, src, path, hcl.InitialPos, allFuncs, nil, &configFile)
	diags = append(diags, d...)
	if diags.HasErrors() {
		fatal := true
		printDiagnostics(parser, diags, fatal)
	}

	return &configFile
}

func printDiagnostics(parser *hclparse.Parser, diags hcl.Diagnostics, fatal bool) {
	color := terminal.IsTerminal(int(os.Stderr.Fd()))
	w, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w = 80
	}

	// TODO(rchapman): line numbers aren't working for commands.  For example, if you have a HCL file like:
	//  1 custom_function "greet" {
	//  2    params = [name]
	//  3    command = <<-EOT
	//  4      n=$(echo ${name} | tr '[a-z]' '[A-Z]')
	//  5      echo ${n}
	//  6  }
	//  there is an error on line 5, because ${} needs to be escaped with double dollar to $${}
	//  the error message will refer to line 2 though (second line in the command attr)
	wr := hcl.NewDiagnosticTextWriter(
		os.Stdout,      // writer to send messages to
		parser.Files(), // the parser's file cache, for source snippets
		uint(w),        // wrapping width
		color,          // generate colored/highlighted output
	)
	wr.WriteDiagnostics(diags)

	if fatal {
		log.Fatal().Msgf("")
	}
}

// funcMap map[string]function.Function, varMap map[string]cty.Value, val interface{}
func decode(parser *hclparse.Parser, src []byte, filename string, start hcl.Pos, funcMap map[string]function.Function,
	varMap map[string]cty.Value, val interface{}) hcl.Diagnostics {

	file, diags := parser.ParseHCL(src, filename)
	if diags.HasErrors() {
		return diags
	}
	if funcMap == nil {
		// empty func map
		funcMap = map[string]function.Function{}
	}
	if varMap == nil {
		// empty variable map
		varMap = map[string]cty.Value{}
	}
	evalCtx := &hcl.EvalContext{
		Functions: funcMap,
		Variables: varMap,
	}

	diags = gohcl.DecodeBody(file.Body, evalCtx, val)
	return diags
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
	/*

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
		//b64PublicJsonKeyset := base64.StdEncoding.EncodeToString(publicKeysetJson)

		// TODO: use HCL libs to modify config.hcl and add pubkey

		//cf.Meta.PublicKeyDefs[pubkeyName] = b64PublicJsonKeyset

		// write up to 64 characters of base64 encoded string to yaml key named pubkeyName

	*/
}
