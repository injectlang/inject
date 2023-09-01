package editfile

import (
	"encoding/base64"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/injectlang/injector"
	"github.com/injectlang/injector/internal/lex"
	"github.com/rs/zerolog/log"
)

type EditConfigFile struct {
	file     *hclwrite.File
	filename string
	src      []byte
	parser   *hclparse.Parser
	parsed   bool
}

func NewEditConfigFile(path string) *EditConfigFile {
	return &EditConfigFile{
		file:     &hclwrite.File{},
		filename: path,
		src:      []byte{},
		parser:   hclparse.NewParser(),
		parsed:   false,
	}
}

func (e *EditConfigFile) AddPublicKey(pubkeyName string, publicJsonKeyset []byte, overwrite bool) hcl.Diagnostics {
	if !e.validatePubkeyName(pubkeyName) {
		msg := "cannot add public key \"%s\" to config file, name of public key must consist of uppercase"
		msg += " letters and numbers"
		log.Fatal().Msgf(fmt.Sprintf(msg, pubkeyName))
	}

	diags := e.parse()
	if diags != nil {
		return diags
	}

	body := e.file.Body()
	pubkeyBlock := body.FirstMatchingBlock("public_key", []string{pubkeyName})
	if pubkeyBlock != nil && !overwrite {
		detail := fmt.Sprintf(" cannot overwrite existing public_key block named %s.", pubkeyName)
		d := hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "cannot overwrite public_key block",
			Detail:   detail,
		}}
		diags = append(diags, d...)
		return diags
	}

	// put new block in body, sort body by sections:
	// - custom_function
	//   - alphabetical
	// - public_key
	//   - alphabetical
	// - context
	//   - alphabetical
	b64PublicJsonKeyset := base64.StdEncoding.EncodeToString(publicJsonKeyset)

	// similar to PEM format, at most 64 chars per line, as a heredoc
	b64formatted := "base64 = <<-EOT\n"
	for i, r := range b64PublicJsonKeyset {
		if i%64 == 0 && i != 0 {
			b64formatted += fmt.Sprintf("\n")
		}
		if i%64 == 0 {
			b64formatted += fmt.Sprintf("    ")
		}
		b64formatted += fmt.Sprintf("%c", r)
	}
	b64formatted += "\n  EOT\n"
	tokens, diags := lex.TokenizeAttrRHS(b64formatted, "base64")
	if diags.HasErrors() {
		return diags
	}

	// add (or overwrite) public_key block named pubkeyName
	block := hclwrite.NewBlock("public_key", []string{pubkeyName})
	body.AppendBlock(block)
	block.Body().SetAttributeRaw("base64", tokens)
	outFile := sortBlocks(e.file)

	write(e.filename, outFile)
	return nil
}

func (e *EditConfigFile) validatePubkeyName(name string) bool {
	re := regexp.MustCompile(`^[A-Z][A-Z0-9]*$`)
	return re.MatchString(name)
}

func (e *EditConfigFile) parse() hcl.Diagnostics {
	if e.parsed {
		// already parsed
		return nil
	}
	src, err := os.ReadFile(e.filename)
	if err != nil {
		// TODO: what's the accepted pattern here?  Do you return this error in diags,
		//       or do you panic right now?
		log.Fatal().Msgf("error reading config file at path %s: %s", e.filename, err)
	}

	file, diags := hclwrite.ParseConfig(src, e.filename, hcl.InitialPos)
	if diags.HasErrors() {
		return diags
	}

	e.file = file
	e.src = src
	e.parsed = true
	return diags
}

func sortBlocks(old *hclwrite.File) *hclwrite.File {
	// collect all the block types (custom_function, public_key, context, everything else)
	var customFuncBlocks []*hclwrite.Block
	var pubkeyBlocks []*hclwrite.Block
	var contextBlocks []*hclwrite.Block
	var otherBlocks []*hclwrite.Block
	for _, b := range old.Body().Blocks() {
		ty := b.Type()
		switch ty {
		case "custom_function":
			customFuncBlocks = append(customFuncBlocks, b)
		case "public_key":
			pubkeyBlocks = append(pubkeyBlocks, b)
		case "context":
			contextBlocks = append(contextBlocks, b)
		default:
			otherBlocks = append(otherBlocks, b)
		}
	}

	// within each block type [customFuncBlocks, pubkeyBlocks, contextBlocks, otherBlocks]
	// alpha sort each type
	sortBlock(customFuncBlocks)
	sortBlock(pubkeyBlocks)
	sortBlock(contextBlocks)
	sortBlock(otherBlocks)

	// create a new hclwrite.File (which has a new hclwrite.Body),
	// then append the newly sorted blocks
	newfile := hclwrite.NewFile()
	newbody := newfile.Body()
	appendBlocks(newbody, customFuncBlocks)
	newbody.AppendNewline()
	appendBlocks(newbody, pubkeyBlocks)
	newbody.AppendNewline()
	appendBlocks(newbody, contextBlocks)
	newbody.AppendNewline()
	appendBlocks(newbody, otherBlocks)

	return newfile
}

// intended to sort a list of blocks within a type.
// for example, sort of list of blocks of type "custom_function"
func sortBlock(x []*hclwrite.Block) {
	sort.SliceStable(x, func(i, j int) bool {
		iBlockName := ""
		if len(x[i].Labels()) > 0 {
			iBlockName = x[i].Labels()[0]
		}
		jBlockName := ""
		if len(x[j].Labels()) > 0 {
			jBlockName = x[j].Labels()[0]
		}
		return iBlockName < jBlockName
	})
}

// intended to append a list of blocks of a certain type.
// for example, append a list of blocks of type "custom_function"
func appendBlocks(body *hclwrite.Body, blocks []*hclwrite.Block) {
	for i, b := range blocks {
		if i != 0 {
			body.AppendNewline()
		}
		body.AppendBlock(b)
	}
}

// ContextNames
//
// return a list of contexts that exist in the config file
func (e *EditConfigFile) ContextNames() ([]string, hcl.Diagnostics) {
	var contexts []string

	diags := e.parse()
	if diags.HasErrors() {
		return contexts, diags
	}

	for _, b := range e.file.Body().Blocks() {
		if b.Type() == "context" {
			labels := b.Labels()
			if len(labels) == 0 {
				continue
			}
			name := labels[0]
			contexts = append(contexts, name)
		}
	}

	return contexts, diags
}

// ExportNames
//
// return a list of export names
func (e *EditConfigFile) ExportNames() ([]string, hcl.Diagnostics) {
	var exportNames []string

	diags := e.parse()
	if diags != nil {
		return exportNames, diags
	}

	// TODO
	//exports := NewExports(e.file, contextName)
	//return exports.GetAll()

	return exportNames, nil
}

// PublicKeyNames
//
// return a list of public_key blocks that exist in the config file
func (e *EditConfigFile) PublicKeyNames() ([]string, hcl.Diagnostics) {
	var pubkeyNames []string

	diags := e.parse()
	if diags != nil {
		return pubkeyNames, diags
	}

	for _, b := range e.file.Body().Blocks() {
		if b.Type() == "public_key" {
			labels := b.Labels()
			if len(labels) == 0 {
				continue
			}
			name := labels[0]
			pubkeyNames = append(pubkeyNames, name)
		}
	}

	return pubkeyNames, diags
}

func (e *EditConfigFile) AddSecret(contextName, exportName, secretVal, pubkeyName string, overwrite bool) hcl.Diagnostics {
	diags := e.parse()
	if diags != nil {
		return diags
	}

	// Take a raw string (that would normally appear in a .inj.hcl file), parse it to extract
	// the tokens. Then set the attribute to the tokens.
	// for example, we have this:
	//
	// context "dev" {
	//   exports = {
	//     DB_NAME = "app1"
	//   }
	// }
	//
	// and we want to add new export named DB_PASSWORD which as an encrypted value.  We want it to
	// end up looking like this:
	//
	// context "dev" {
	//   exports = {
	//     DB_NAME = "app1"
	//     DB_PASSWORD = decrypt("DEV2022", "AQBdqk1SXCc0Yd26m1+XVhs1LrMrhFTf473Hv7bbTS9getqBAYFkSUwRjt0FqcFHyRQMLTwLXVP+I/oWIOczpSqFMg==")
	//   }
	// }
	pubkey, diags := getPubkey(e.file.Body(), pubkeyName)
	if diags.HasErrors() {
		return diags
	}
	encryptor := injector.NewEncryptor(string(pubkey))
	cipherbytes, err := encryptor.Encrypt([]byte(secretVal), nil)
	if err != nil {
		d := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Could not encrypt",
			Detail:   fmt.Sprintf("Could not encrypt secret using public_key \"%s\"", pubkeyName),
		}
		diags = append(diags, d)
		return diags
	}
	b64ciphertext := base64.StdEncoding.EncodeToString(cipherbytes)

	exports := NewExports(e.file, contextName)
	diags = exports.SetEncryptedValue(exportName, pubkeyName, b64ciphertext, overwrite)
	if diags.HasErrors() {
		return diags
	}

	write(e.filename, e.file)
	return nil
}

// Look up a block public_key named `name`.
// Take the `base64` attribute from the block.
// Base64 decode it, convert to []byte,
// return the byte array.
func getPubkey(body *hclwrite.Body, pubkeyName string) ([]byte, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	block := body.FirstMatchingBlock("public_key", []string{pubkeyName})
	if block == nil {
		d := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid public_key block",
			Detail:   fmt.Sprintf("A public_key block named %s cannot be found", pubkeyName),
		}
		diags = append(diags, d)
		return nil, diags
	}
	tokens := block.Body().GetAttribute("base64").Expr().BuildTokens(nil)
	var cleansedTokens hclwrite.Tokens
	for _, t := range tokens {
		if t.Type == hclsyntax.TokenOHeredoc {
			// skip over OpenHeredoc tokens like `<<-EOT`
			continue
		}
		if t.Type == hclsyntax.TokenCHeredoc {
			// skip over CloseHeredoc tokens like `EOT`
			continue
		}
		cleansedTokens = append(cleansedTokens, t)
	}
	tokensStr := string(cleansedTokens.Bytes())
	tokensStr = strings.ReplaceAll(tokensStr, "\n", "")
	pubkeyb64 := strings.ReplaceAll(tokensStr, " ", "")
	pubkeyBytes, err := base64.StdEncoding.DecodeString(pubkeyb64)
	if err != nil {
		detail := fmt.Sprintf("While processing public_key \"%s\", ", pubkeyName)
		detail += fmt.Sprintf("could not base64 decode \"%s\": %s", pubkeyb64, err)
		d := &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot base64 decode",
			Detail:   detail,
		}
		diags = append(diags, d)
		return nil, diags
	}
	return pubkeyBytes, diags
}

func write(filename string, outFile *hclwrite.File) {
	outFileInfo, err := os.Stat(filename)
	if err != nil {
		log.Fatal().Msgf("cannot stat output file %s: %s", filename, err)
	}
	out, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, outFileInfo.Mode())
	if err != nil {
		log.Fatal().Msgf("cannot open output file %s: %s", filename, err)
	}
	_, err = outFile.WriteTo(out)
	if err != nil {
		log.Fatal().Msgf("cannot write to output file %s: %s", filename, err)
	}
}
