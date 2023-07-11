package main

import (
	// "io/ioutil"
	"log"
	//	"os"

	//	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
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

func main() {
	//	yml, err := ioutil.ReadFile("config.yml.tmpl")
	//	if err != nil {
	//		log.Panicf("ReadFile(): %+v", err)
	//	}
	//	cf := ConfigFile{}
	//	err = yaml.UnmarshalWithOptions([]byte(yml), &cf)
	//
	//	if err != nil {
	//		log.Panicf("unmarshal: %+v", yaml.FormatError(err, true, true))
	//	}

	/*path, err := yaml.PathString("$")
	if err != nil {
		log.Panicf("PathString(): %+v", yaml.FormatError(err, true, true))
	}

	f, err := os.Open("config.yml.tmpl")
	if err != nil {
		log.Panicf("Open(): %+v", err)
	}
	astNode, err := path.ReadNode(f)
	if err != nil {
		log.Panicf("ReadNode(): %+v", yaml.FormatError(err, true, true))
	}*/
	astFile, err := parser.ParseFile("config.yml", 0)
	if err != nil {
		log.Panicf("ParseFile(): %+v", err)
	}

	for _, doc := range astFile.Docs {
		log.Printf("doc: %T = %+v\n", doc, doc)

		//		node := doc.Body
		var v Visitor
		ast.Walk(&v, doc.Body)
		//		log.Printf("========\n")
		//		log.Printf("%T = %+v\n", node, node)
	}
}

type Visitor struct {
}

func (v *Visitor) Visit(node ast.Node) ast.Visitor {
	tk := node.GetToken()
	yamlpathstr := node.GetPath() // e.g. "$._meta"
	log.Printf("Visit: %T = %+v\n", node, node)
	log.Printf("       %T = %+v\n", tk, tk)
	log.Printf("       %T = %+v\n", yamlpathstr, yamlpathstr)
	log.Printf("\n")
	tk.Prev = nil
	tk.Next = nil
	return v
}
