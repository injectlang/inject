// Based on userfunc extension in the HCL library,
// at https://github.com/hashicorp/hcl/tree/main/ext/userfunc
//
// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package customfunc

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/rs/zerolog/log"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"os"
	"os/exec"
	"strings"
)

const BLOCK_TYPE = "custom_function"

var funcBodySchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name:     "params",
			Required: true,
		},
		{
			Name:     "command",
			Required: true,
		},
	},
}

func decodeCustomFunctions(parser *hclparse.Parser, src []byte, filename string, contextFunc ContextFunc) (funcs map[string]function.Function, remain hcl.Body, diags hcl.Diagnostics) {
	file, diags := parser.ParseHCL(src, filename)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       BLOCK_TYPE,
				LabelNames: []string{"name"},
			},
		},
	}

	content, remain, diags := file.Body.PartialContent(schema)
	if diags.HasErrors() {
		return nil, remain, diags
	}

	ni := newNonInterpolatedCmdStr(src, filename)

	funcs = make(map[string]function.Function)
Blocks:
	for _, block := range content.Blocks.OfType(BLOCK_TYPE) {
		funcName := block.Labels[0]
		funcContent, funcDiags := block.Body.Content(funcBodySchema)

		diags = append(diags, funcDiags...)
		if funcDiags.HasErrors() {
			continue
		}

		paramsExpr := funcContent.Attributes["params"].Expr
		paramExprs, paramsDiags := hcl.ExprList(paramsExpr)
		if paramsDiags.HasErrors() {
			continue
		}

		paramNames := []string{}
		for _, paramExpr := range paramExprs {
			paramName := hcl.ExprAsKeyword(paramExpr)
			if paramName == "" {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid param element",
					Detail:   "Each parameter name must be an identifier.",
					Subject:  paramExpr.Range().Ptr(),
				})
				continue Blocks
			}
			paramNames = append(paramNames, paramName)
		}

		allParams := []function.Parameter{}
		for _, p := range paramNames {
			param := function.Parameter{
				Name: p,
				Type: cty.String,
			}
			allParams = append(allParams, param)
		}

		customFunc := function.New(&function.Spec{
			Params:       allParams,
			Type:         function.StaticReturnType(cty.String),
			RefineResult: RefineNonNull,
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				// param1 := args[0].AsString()
				// param2 := args[1].AsString()
				funcRange := funcContent.Attributes["command"].Range
				cmdRecord, diags := ni.GetCommandRecord(funcName, funcRange)
				if diags.HasErrors() {
					// Smuggle the diagnostics out via the error channel, since
					// a diagnostics sequence implements error. Caller can
					// type-assert this to recover the individual diagnostics
					// if desired.
					return cty.DynamicVal, diags
				}
				rawCmdBytes := []byte(cmdRecord.Value)
				expr, diags := hclsyntax.ParseTemplate(rawCmdBytes, filename, cmdRecord.SrcRange.Start)
				if diags.HasErrors() {
					return cty.DynamicVal, diags
				}
				// The cty function machinery guarantees that we have at least
				// enough args to fill all of our params.
				varMap := make(map[string]cty.Value, len(allParams))
				for i, param := range allParams {
					varMap[param.Name] = args[i]
				}
				evalCtx := &hcl.EvalContext{
					Functions: map[string]function.Function{},
					Variables: varMap,
				}

				interpolated, diags := expr.Value(evalCtx)
				if diags.HasErrors() {
					return cty.DynamicVal, diags
				}
				cmdStr := interpolated.AsString()

				shell := os.Getenv("SHELL")
				if shell == "" {
					shell = "/bin/sh"
				}
				log.Debug().Msgf("interpolated cmdStr: [%s]\n", cmdStr)
				cmd := exec.Command(shell, "-c", cmdStr)
				out, err := cmd.CombinedOutput() // on success, Output() returns stdout. On error, Output() returns stderr
				exitCode := -1
				if cmd != nil && cmd.ProcessState != nil {
					exitCode = cmd.ProcessState.ExitCode()
				}
				log.Debug().Msgf("exec \"%s\" returned %d", cmdStr, exitCode)
				if err != nil || exitCode != 0 {
					detail := fmt.Sprintf("Command \"%s\" defined by the custom_function \"%s\" returned non-zero (%d): %s",
						cmdRecord.Value, funcName, exitCode, err)
					detail += fmt.Sprintf(", stdout_stderr=%s", out)
					d := hcl.Diagnostics{{
						Severity: hcl.DiagError,
						Summary:  "Cannot execute command",
						Detail:   detail,
					}}
					diags = append(diags, d...)
					return cty.DynamicVal, diags
				}
				outStr := strings.TrimSuffix(string(out), "\n")
				return cty.StringVal(outStr), nil
			},
		})

		funcs[funcName] = customFunc
	}

	return funcs, remain, diags
}

func RefineNonNull(b *cty.RefinementBuilder) *cty.RefinementBuilder {
	return b.NotNull()
}
