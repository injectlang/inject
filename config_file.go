package injector

// This file handles parsing a .inj config file into a struct
// called ConfigFile

import (
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/injectlang/injector/customfunc"
	"github.com/rs/zerolog/log"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// There are a few sections in a config.inj.hcl file:
//
// custom_function "greet" {
// }
//
// custom_function "getCfnOutput" {
// }
//
// public_key "DEV2022" {
// }
//
// public_key "STAGING2021" {
// }
//
// public_key "PROD2023" {
// }
//
// context "dev" {
// }
//
// context "staging" {
// }
//
// context "prod" {
// }
//

// ConfigFile
//
// We are only concerned with the context blocks in this struct
// Decoding/encoding of other blocks is done manually in the other
// libraries.
type ConfigFile struct {
	Contexts Contexts `hcl:"context,block"`
	Remain   hcl.Body `hcl:",remain"`
}

type Contexts []Context

type Context struct {
	Name    string            `hcl:"name,label"`
	Vars    map[string]string `hcl:"vars,optional"`
	Exports map[string]string `hcl:"exports"` // environment variables used by application
}

const DEFAULT_CONFIG_FILE_PATH = "config.inj.hcl"

func NewConfigFile(path string) *ConfigFile {
	if path == "" {
		path = DEFAULT_CONFIG_FILE_PATH
	}

	src, err := os.ReadFile(path)
	if err != nil {
		log.Panic().Msgf("error reading config file at path %s: %s", path, err)
	}

	// TODO(rchapman): parsing doesn't belong in a New() function
	//                 split it out to `func (cf *ConfigFile) Parse()`
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
		PrintDiagnostics(parser, diags, fatal)
	}

	return &configFile
}

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

// TODO(rchapman): write func that prints rendered .inj.hcl file
//                 be sure to mask out anything that was rendered by decrypt()
//                 if len of secret (plaintext) is:
//                   len <= 5, print first and last char
//                   len > 5, print first two and last two chars
//                 this will be printed whenever injector or injectord is running in debug mode
