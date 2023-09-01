package editfile

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/injectlang/injector/internal/parse"
)

type Exports struct {
	file                *hclwrite.File
	contextName         string
	parsed              bool
	parsedExportRecords parse.ExportRecords
	parsedDiags         hcl.Diagnostics
}

func NewExports(file *hclwrite.File, contextName string) *Exports {
	return &Exports{
		file:        file,
		contextName: contextName,
	}
}

// Given a file (of type hclwrite.File), return the first context
// block named e.contextBlock
//
// for example, given
//
//	context "dev" {}
//	context "staging" {}
//
// and
//
//	e := NewExports(file, "staging")
//
// then
//
//	e.findContext()
//
// would return
//
//	context "staging" {}
func (e *Exports) findContext() (*hclwrite.Block, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	contextBlock := e.file.Body().FirstMatchingBlock("context", []string{e.contextName})
	if contextBlock == nil {
		d := hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "could not find context block",
			Detail:   fmt.Sprintf("The context block %s cannot be found.", e.contextName),
		}}
		diags = append(diags, d...)
	}

	return contextBlock, diags
}

// within a context block, return the exports object
//
// for example, given
//
//	context "dev" {
//	  exports = {
//	    DB_USERNAME = "db"
//	  }
//	}
//
// return
//
//	exports = {
//	  DB_USERNAME = "db"
//	}
func (e *Exports) findExportsAttr() (*hclwrite.Attribute, hcl.Diagnostics) {
	contextBlock, diags := e.findContext()
	if diags.HasErrors() {
		return nil, diags
	}

	exportsAttr := contextBlock.Body().GetAttribute("exports")
	if exportsAttr == nil {
		d := hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "could not find exports object",
			Detail:   fmt.Sprintf("The object \"exports\" cannot be found in the \"%s\" context block.", e.contextName),
		}}
		diags = append(diags, d...)
	}

	return exportsAttr, diags
}

// Exists
//
// Determine if an export exists
// Time complexity: O(n)
func (e *Exports) Exists(exportName string) (bool, hcl.Diagnostics) {
	allExports, diags := e.GetAll()
	if diags.HasErrors() {
		return false, diags
	}

	for _, export := range allExports {
		if strings.TrimSpace(export.Name) == exportName {
			return true, nil
		}
	}

	return false, nil
}

// GetAll
//
// return a list of exports that exist in a context block
func (e *Exports) GetAll() (parse.ExportRecords, hcl.Diagnostics) {
	if e.parsed {
		return e.parsedExportRecords, e.parsedDiags
	}
	var diags hcl.Diagnostics

	exportsAttr, diags := e.findExportsAttr()
	if diags.HasErrors() {
		return nil, diags
	}

	tokens := exportsAttr.Expr().BuildTokens(nil)
	exportRecords, diags := parse.Parse(tokens)

	// even if parse returned errors, cache it.  No point in re-parsing later.
	e.parsedExportRecords = exportRecords
	e.parsedDiags = diags
	e.parsed = true

	return exportRecords, diags
}

// SetEncryptedValue adds an export
//
// If an export already exists and overwrite is true, existing
// export will be overwritten.  If an export exists and
// overwrite if false, an error will be returned.
//
// Example:
//
//	e := NewExports("dev")
//	e.SetEncryptedValue("DB_PASSWORD", "DEV2022", "<encryptedStr>", true)
//
// would produce:
//
//	context "dev" {
//	  exports = {
//	    DB_PASSWORD = decrypt("DEV2022", "<encrypted_string>")
//	  }
//	}
func (e *Exports) SetEncryptedValue(exportName, pubkeyName, encrypted string, overwrite bool) hcl.Diagnostics {
	// Find the context, then the exports attr
	// from that point, we have Tokens.
	// Walk the tokens, if the export already exists
	// just need to set the value.
	// Uf the export does not exist, find the alphabetical location
	// for exportName, insert new tokens needed.

	if !validateName(exportName) {
		msg := "The export named \"%s\" in context block \"%s\" cannot be added/overwritten in config file."
		msg += " An export must be a valid environment variable name."
		diags := hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "invalid export name",
			Detail:   fmt.Sprintf(msg, exportName, e.contextName),
		}}
		return diags
	}

	exportRecords, diags := e.GetAll()
	if diags.HasErrors() {
		return diags
	}

	exists, diags := e.Exists(exportName)
	if diags.HasErrors() {
		return diags
	}

	if exists && !overwrite {
		d := hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "cannot overwrite export",
			Detail:   fmt.Sprintf("export \"%s\" already exists, and overwrite not requested", exportName),
		}}
		diags = append(diags, d...)
		return diags
	}

	// add (or overwrite) export record
	newVal := fmt.Sprintf(" decrypt(\"%s\", \"%s\")", pubkeyName, encrypted)
	found := false
	for _, r := range exportRecords {
		fmt.Printf("r.Name: %s, exportName: %s, r.Name == exportName ? %v\n", r.Name, exportName, r.Name == exportName)
		// handle overwriting existing record
		if r.Name == exportName {
			r.SetValue(newVal)
			found = true
		}
	}

	// existing record not found, append to record list
	if !found {
		exportRecords = append(exportRecords, &parse.ExportRecord{
			Name:  exportName,
			Value: newVal,
		})
	}

	contextBlock, diags := e.findContext()
	if diags.HasErrors() {
		return diags
	}

	newTokens, diags := exportRecords.Tokens()
	if diags.HasErrors() {
		return diags
	}

	fmt.Printf("newExportTokens:\n%s\n", exportRecords.String())

	contextBlock.Body().SetAttributeRaw("exports", newTokens)

	return nil
}

func validateName(exportName string) bool {
	re := regexp.MustCompile(`^[A-Z_]{1,}[A-Z0-9_]+$`)
	return re.MatchString(exportName)
}
