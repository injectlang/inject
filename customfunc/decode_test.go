// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package customfunc

import (
	"fmt"
	"github.com/hashicorp/hcl/v2/hclparse"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	_ "github.com/injectlang/injector/log"
	_ "github.com/rs/zerolog/log"
	"github.com/zclconf/go-cty/cty"
)

func TestDecodeUserFunctions(t *testing.T) {
	tests := []struct {
		src       string
		testExpr  string
		baseCtx   *hcl.EvalContext
		want      cty.Value
		diagCount int
	}{
		{
			`
custom_function "greet" {
  params = [name]
  command = "echo \"Hello, ${name}.\""
}
`,
			`greet("Peter")`,
			nil,
			cty.StringVal("Hello, Peter."),
			0,
		},
		{
			`
custom_function "greet_heredoc" {
  params = [name]
  command = <<EOT
echo "Hello, ${name}."
EOT
}
`,
			`greet_heredoc("Peter")`,
			nil,
			cty.StringVal("Hello, Peter."),
			0,
		},
		{
			`
custom_function "greet_heredoc2" {
  params = [name]
  command = <<-EOT
    echo "Hello, ${name}."
  EOT
}
`,
			`greet_heredoc2("Peter")`,
			nil,
			cty.StringVal("Hello, Peter."),
			0,
		},
		{
			`
custom_function "greet_multiline_heredoc" {
  params = [name]
  command = <<-EOT
    echo "Hello, ${name}."
    echo "Hello again, ${name}."
  EOT
}
`,
			`greet_multiline_heredoc("Peter")`,
			nil,
			cty.StringVal("Hello, Peter.\nHello again, Peter."),
			0,
		},
		{
			`
custom_function "greet_multiline_pipe_heredoc" {
  params = [name]
  command = <<-EOT
    (echo "Hello, ${name}."
     echo "Hello again, ${name}.") |
    grep again
  EOT
}
`,
			`greet_multiline_pipe_heredoc("Peter")`,
			nil,
			cty.StringVal("Hello again, Peter."),
			0,
		},
		{
			`
custom_function "greet" {
  params = [name]
  command = "echo \"Hello, ${name}.\""
}
`,
			`greet()`,
			nil,
			cty.DynamicVal,
			1, // missing value for "name"
		},
		{
			`
custom_function "greet" {
  params = [name]
  command = "echo \"Hello, ${name}.\""
}
`,
			`greet("Peter", "extra")`,
			nil,
			cty.DynamicVal,
			1, // too many arguments
		},
		{
			`
custom_function "missing_command" {
  params = [name, age]
  commnd = "echo Hi ${name}, I hear you are ${age} years old."
}
`,
			`missing_command("Peter", 20)`,
			nil,
			cty.DynamicVal,
			3, // no parameter named "command"
		},
		{
			`
custom_function "missing_var" {
  params = []
  command = "echo \"${nonexist}\""
}
`,
			`missing_var()`,
			nil,
			cty.DynamicVal,
			1, // no variable named "nonexist"
		},
		{
			`
custom_function "missing_params" {
  parrams = [val]
  command = "echo \"${val}\""
}
`,
			`null`,
			nil,
			cty.NullVal(cty.DynamicPseudoType),
			2, // missing attribute "params", and unknown attribute "parrams"
		},
		{
			`
custom_function "greet_empty_params" {
  params = []
  command = <<-EOT
    echo "Hello."
  EOT
}
`,
			`greet_empty_params()`,
			nil,
			cty.StringVal("Hello."),
			0,
		},
		{
			`
custom_function "missing_var_heredoc" {
  params = []
  command = <<EOT
echo "${nonexist}"
EOT
}
`,
			`missing_var_heredoc()`,
			nil,
			cty.DynamicVal,
			1, // no variable named "nonexist"
		},
		{
			`
custom_function "failed_command" {
  params = []
  command = "exit 1"
}
`,
			`failed_command()`,
			nil,
			cty.DynamicVal,
			1, // cannot execute command, returned 1
		},
		{
			`
custom_function "failed_command_with_stderr" {
  params = []
  command = <<-EOT
    echo "There was a problem getting the CFN output named blah" >&2
    exit 1
  EOT
}
`,
			`failed_command_with_stderr()`,
			nil,
			cty.DynamicVal,
			1, // cannot execute command, returned 1
		},
		{
			`
custom_function "stderr_test" {
  params = [name]
  command = <<-EOT
    echo "Hello ${name}! stderr" 1>&2
    exit 0
  EOT
}
`,
			`stderr_test("Peter")`,
			nil,
			cty.StringVal("Hello Peter! stderr"),
			0,
		},
		{
			`
custom_function "greet" {
  params = [name]
  command = "echo ${name}"
`,
			`nonexist_invalid_syntax("Peter"`,
			nil,
			cty.DynamicVal,
			4,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			src := []byte(test.src)
			parser := hclparse.NewParser()
			f, diags := parser.ParseHCL(src, "config")
			//f, diags := hclsyntax.ParseConfig(src, "config", hcl.Pos{Line: 1, Column: 1})
			if f == nil || f.Body == nil {
				t.Fatalf("got nil file or body")
			}

			funcs, _, funcsDiags := decodeCustomFunctions(parser, src, "function", func() *hcl.EvalContext {
				return test.baseCtx
			})
			diags = append(diags, funcsDiags...)

			expr, exprParseDiags := hclsyntax.ParseExpression([]byte(test.testExpr), "testexpr", hcl.Pos{Line: 1, Column: 1})
			diags = append(diags, exprParseDiags...)
			if expr == nil {
				t.Fatalf("parsing test expr returned nil")
			}

			got, exprDiags := expr.Value(&hcl.EvalContext{
				Functions: funcs,
			})
			diags = append(diags, exprDiags...)

			if len(diags) != test.diagCount {
				t.Errorf("wrong number of diagnostics %d; want %d", len(diags), test.diagCount)
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}

			if !got.RawEquals(test.want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}
