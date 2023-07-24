package main

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
	//	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/ryanchapman/config-container/sidecar"
	_ "github.com/ryanchapman/config-container/sidecar/log"
)

func main() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	configFilePath := "config.hcl"
	cf := sidecar.NewConfigFile(configFilePath)

	fmt.Printf("Configuration is %+v", cf)

	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(cf, f.Body())
	fmt.Printf("%s", f.Bytes())
}
