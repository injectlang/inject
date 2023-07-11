package sidecar

// This file deals with parsing a config file as a golang template.
//
//
// Templating is done in three passes:
//
// Pass 1: remove comments (lines start with '#')
//
// Pass 2: the template file is ingested and any templates are removed.
//         so any instances of "{{.*}}" are removed.  This is so that
//         data in the config file can be referenced in pass 2.
//
//         For example, given the config.yml.tmpl file:
//
//         meta:
//           public_keys:
//             dev2020: |
//               aGVsbG93b3JsZAo=
//           globals:
//             db_username: "app1"
//
//         environments:
//           dev:
//             - DB_USERNAME: "{{ .meta.globals.db_username }}"
//           staging:
//             - DB_USERNAME: "{{ .meta.globals.db_username }}"
//           production:
//             - DB_USERNAME: "{{ .meta.globals.db_username }}"
//
//        You'll see that we can reference yaml values in the golang template
//        "{{ }}" construct.
//
// Pass 3: the template file is ingested again as a golang template.
//         Any instances of "{{ }}" are interpreted.
//

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"
)

type ConfigTemplate struct {
	Original    []byte
	Interpreted ConfigFile
	funcMap     template.FuncMap
}

func NewConfigTemplateFromFile(configFilePath string, templateFuncMap template.FuncMap) *ConfigTemplate {
	yaml, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Panicf("could not read template file %s: %+v", configFilePath, err)
	}
	return NewConfigTemplate(yaml, templateFuncMap)
}

func NewConfigTemplate(configFileYaml []byte, templateFuncMap template.FuncMap) *ConfigTemplate {
	pass1 := pass1(configFileYaml)
	pass2 := pass2(pass1)
	pass3 := pass3(pass1, pass2, templateFuncMap)

	ctFinal := &ConfigTemplate{
		Original:    configFileYaml,
		Interpreted: *pass3,
		funcMap:     templateFuncMap,
	}
	return ctFinal
}

func logLevel() string {
	return strings.ToLower(os.Getenv("LOG_TEMPLATE"))
}

// remove comments
func pass1(configFileBytes []byte) []byte {
	yaml := string(configFileBytes)
	if logLevel() == "debug" {
		log.Printf("PASS1: before removing comments, configFileYaml=%v", yaml)
	}
	re1 := regexp.MustCompile("(?m)^#.*$\n")
	yaml = re1.ReplaceAllString(yaml, "")
	re2 := regexp.MustCompile("(?m)#.*$")
	yaml = re2.ReplaceAllString(yaml, "")
	if logLevel() == "debug" {
		log.Printf("PASS1: after removing comments, configFileYaml=%v", yaml)
	}
	return []byte(yaml)
}

func pass2(configFileBytes []byte) *ConfigFile {
	// we need to walk the ConfigFile structure.  To avoid reflection, and coupling
	// (this file needing to know about the struct of ConfigFile), change ConfigFile
	// back into yaml, use a regex to remove "{{.*}}", then change the yaml
	// into ConfigFile
	yaml := string(configFileBytes)
	if logLevel() == "debug" {
		log.Printf("PASS2: before removing handlebars, configFileYaml=%v", yaml)
	}
	re := regexp.MustCompile(`{{.*}}`)
	yaml = re.ReplaceAllString(yaml, "")
	if logLevel() == "debug" {
		log.Printf("PASS2: after removing handlebars, configFileYaml=%v", yaml)
	}
	cfPass2 := NewConfigFileFromYaml(yaml)
	return cfPass2
}

func pass3(configFileBytes []byte, pass2 *ConfigFile, funcs template.FuncMap) *ConfigFile {
	if logLevel() == "debug" {
		log.Printf("PASS3: before templating, configFile=%v", *pass2)
	}

	configFileYaml := string(configFileBytes)
	configTmpl, err := template.New("config").Funcs(funcs).Parse(configFileYaml)
	if err != nil {
		log.Panicf("PASS3: error parsing template: err=%s", err)
	}

	buf := new(bytes.Buffer)
	err = configTmpl.Execute(buf, *pass2)
	if err != nil {
		log.Panicf("PASS3: error executing template: err=%s", err)
	}

	parsedBytes, err := io.ReadAll(buf)
	if err != nil {
		log.Panicf("PASS3: error reading bytes from template buffer, err=%s", err)
	}
	cfPass3 := NewConfigFileFromYaml(string(parsedBytes))

	if logLevel() == "debug" {
		log.Printf("PASS3: after templating, configFile=%v", *cfPass3)
	}
	return cfPass3
}
