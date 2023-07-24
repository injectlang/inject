package customfunc

import (
	"bytes"
	"fmt"
	"github.com/rs/zerolog/log"
	"strings"
)

// Provide a way to get a non-interpolated "command" from a custom_function definition.
//
// For example, given this hcl:
//
//	custom_function "name" {
//	  params = [a,b,c]
//	  command = <<-EOT
//	      echo "${a} ${b} ${c}"
// 	  EOT
//	}
//
// we need to grab
// 	      echo "${a} ${b} ${c}"
//
// There has got to be a better way to do this, but I don't know enough about HCL yet.
//
import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type nonInterpolatedCmdStr struct {
	file           *hclwrite.File
	filename       string
	searched       bool
	commandRecords map[string]commandRecord
	diags          hcl.Diagnostics
}

type commandRecord struct {
	Value    string
	SrcRange hcl.Range
}

func newNonInterpolatedCmdStr(src []byte, filename string) *nonInterpolatedCmdStr {
	f, d := hclwrite.ParseConfig(src, filename, hcl.InitialPos)

	n := nonInterpolatedCmdStr{
		file:           f,
		filename:       filename,
		searched:       false,
		commandRecords: map[string]commandRecord{},
		diags:          d,
	}
	return &n
}

func (ni *nonInterpolatedCmdStr) GetCommandRecord(funcName string, funcRange hcl.Range) (commandRecord, hcl.Diagnostics) {
	commandRecords, diags := ni.searchForCommandRecords(funcRange)
	var cmdRec commandRecord
	cmdRec, ok := commandRecords[funcName]
	if !ok {
		d := hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "could not find custom function",
			Detail:   fmt.Sprintf("The custom_function %q cannot be found.", funcName),
		}}
		diags = append(diags, d...)
		return commandRecord{}, diags
	}
	return cmdRec, diags
}

func (ni *nonInterpolatedCmdStr) searchForCommandRecords(funcRange hcl.Range) (map[string]commandRecord, hcl.Diagnostics) {
	if !ni.searched {
		commandRecords, diags := ni.decodeCommandRecords(ni.file.Body(), funcRange)
		ni.commandRecords = commandRecords
		ni.diags = diags
		ni.searched = true
	}
	return ni.commandRecords, ni.diags
}

// Given any block that looks like
//
//		custom_function "name" {
//		  params = [a,b,c]
//		  command = <<-EOT
//		      echo "${a} ${b} ${c}"
//	      EOT
//		}
//
// this function will return the non-interpolated string from command.
//
// Without this, if you had a custom_function like this:
//
//	    custom_function "name" {
//		  params = [region]
//		  command = "echo ${region}"
//		}
//
// the top-level file would fail to decode, because the region variable isn't yet known.
//
// in the above example, this function (GetCommandStr) would return the string
// "echo ${region}"
func (ni *nonInterpolatedCmdStr) decodeCommandRecords(body *hclwrite.Body, funcRange hcl.Range) (map[string]commandRecord, hcl.Diagnostics) {
	return decodeBlocks(body, funcRange)
}

func decodeBlocks(body *hclwrite.Body, funcRange hcl.Range) (map[string]commandRecord, hcl.Diagnostics) {
	// process all customfuncs and return map[string]string
	// e.g. map[cfnOutput] = "echo ${region}"
	// so we don't have to do all this work (call into these functions)
	// for every custom func
	cmdRecords := map[string]commandRecord{}
	var diags hcl.Diagnostics
	var inBlocks []string

	blocks := body.Blocks()

	for _, block := range blocks {
		if block.Type() != BLOCK_TYPE {
			continue
		}
		blockLabels := block.Labels()
		blockLabel0 := ""
		if len(blockLabels) > 0 {
			blockLabel0 = blockLabels[0]
		}
		inBlocks := append(inBlocks, block.Type())
		cmdStr, startOffset, endOffset, attrDiags := decodeAttrs(block.Body(), funcRange.Filename, inBlocks, blockLabel0)
		diags = append(diags, attrDiags...)
		cmdRecords[blockLabel0] = commandRecord{
			Value: cmdStr,
			SrcRange: hcl.Range{
				Filename: funcRange.Filename,
				Start: hcl.Pos{
					Line: funcRange.Start.Line + startOffset,
				},
				End: hcl.Pos{
					Line: funcRange.End.Line + endOffset,
				},
			},
		}
	}

	return cmdRecords, diags
}

// given a block, if it's a custom_function, walk the attributes to find the
// attr named "command".  Grab the tokens for that attr, and convert back
// to a raw string.  For example, given this block:
//
//	custom_function "cfnOutput" {
//	  params = [region]
//	  command = "echo ${region}"
//	}
//
// we want to return "echo ${region}"
func decodeAttrs(body *hclwrite.Body, filename string, inBlocks []string, blockLabel0 string) (string, int, int, hcl.Diagnostics) {
	// code from https://github.com/apparentlymart/terraform-clean-syntax/tree/master
	// Mozilla Public License, version 2.0
	//
	// see also discussion at
	// https://discuss.hashicorp.com/t/parse-hcl-treating-variables-or-functions-as-raw-strings-hashicorp-hcl/5859/2
	attrs := body.Attributes()
	for name, attr := range attrs {
		tokens := attr.Expr().BuildTokens(nil)
		if len(inBlocks) == 1 {
			inBlock := inBlocks[0]
			if inBlock == "custom_function" && name == "command" {
				cmdStr, startOffset, endOffset := commandTokensToStr(tokens)
				cmdStr = unescapeQuotes(cmdStr)
				return cmdStr, startOffset, endOffset, nil
			}
		}
	}

	diags := hcl.Diagnostics{{
		Severity: hcl.DiagError,
		Summary:  "could not find custom function",
		Detail:   fmt.Sprintf("In custom_function %q, could not find attribute \"command\".", blockLabel0),
	}}

	return "", 0, 0, diags
}

func unescapeQuotes(s string) string {
	return strings.ReplaceAll(s, `\"`, `"`)
}

// Using a tokenization of "command", remove some token types
// (e.g. heredoc) and return a string representation.  If any
// tokens are removed, offset will be set to +1, -1, etc.
//
// returns: commandStr string, startOffset int, endOffset int
func commandTokensToStr(tokens hclwrite.Tokens) (string, int, int) {
	startOffset := 0
	endOffset := 0
	// if present, remove:
	//   open/close heredoc tokens
	//   open/close quote tokens
	var cleansedTokens hclwrite.Tokens
	for _, token := range tokens {
		ttype := token.Type
		switch ttype {
		case hclsyntax.TokenOHeredoc:
			startOffset = startOffset + 1
			continue
		case hclsyntax.TokenCHeredoc:
			endOffset = endOffset - 1
			continue
		case hclsyntax.TokenOQuote:
			continue
		case hclsyntax.TokenCQuote:
			continue
		}
		cleansedTokens = append(cleansedTokens, token)
	}
	for i, t := range cleansedTokens {
		log.Debug().Msgf("cleansedToken[%d] (%s) = %s", i, t.Type, string(t.Bytes))
	}
	var buf bytes.Buffer
	cleansedTokens.WriteTo(&buf)
	cmdBytes := bytes.TrimSpace(buf.Bytes())
	return string(cmdBytes), startOffset, endOffset
}
