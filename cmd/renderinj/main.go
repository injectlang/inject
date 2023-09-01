package main

// Render a config.inj file like what would happen at container/VM startup,
// which is useful for debugging.  you must provide the same env vars that would
// be needed by injector or injectord.

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/injectlang/injector"
	_ "github.com/injectlang/injector/log"
	//	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

func main() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// TODO(rchapman): add usage

	configFilePath := "config.inj.hcl"
	cf := injector.NewConfigFile(configFilePath)

	fmt.Printf("Configuration is %+v", cf)

	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(cf, f.Body())
	fmt.Printf("%s", f.Bytes())
}
