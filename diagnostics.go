package injector

import (
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh/terminal"
)

func PrintDiagnostics(parser *hclparse.Parser, diags hcl.Diagnostics, fatal bool) {
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
