//
// Parse exports
//
// Given an array of tokens ([]*hclwrite.Token aka hclwrite.Tokens)
// representing an exports object, parse and provide access to
// key/value pairs.
//
// WARNING: this is low level and does not verify that the tokens passed in
//          represent valid syntax.  You need to do this yourself using something like:
//
// WARNING: this does not check for duplicate keys (.Name field).
//          You need to do this yourself.
//
//   src, err := os.ReadFile(filename)
//	 if err != nil {
//	   log.Fatal().Msgf("error reading config file at path %s: %s", filename, err)
//	 }
//
//   file, diags := hclwrite.ParseConfig(src, filename, hcl.InitialPos)
//	 if diags.HasErrors() { ... }
//
//
// Obtaining tokens...
//
// Given a sample.inj.hcl file with contents:
//
//   context "dev" {
//     exports = {
//       DB_USERNAME = "db"
//       DB_PASSWORD = decrypt("DEV", "c3VwZXJTZWNyZXRQcm9k")
//     }
//   }
//
// We can obtain the tokens with:
//
//   src, err := os.ReadFile("sample.inj.hcl")
//   if err != nil {
//     ...
//   }
//   file, diags := hclwrite.ParseConfig(src, "sample.inj.hcl", hcl.InitialPos)
//   if diags.HasErrors() {
//     ...
//   }
//   file := hclwrite.NewFile(...)
//   block := file.Body().FirstMatchingBlock("context", []string{"dev"})
//   attr := block.Body().GetAttribute("exports")
//   tokens := attr.Expr().BuildTokens(nil)
//
//

package parse

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/injectlang/injector/internal/lex"
)

type ExportRecord struct {
	Name  string
	Value string
}

func (e *ExportRecord) String() string {
	s := ""
	s += e.Name
	if e.Value != "" {
		s += " ="
		s += e.Value
		// comments have \n included in .Name, so don't have to add it here
		s += "\n"
	}
	return s
}

func (e *ExportRecord) Tokens() (hclwrite.Tokens, hcl.Diagnostics) {
	tokens, diags := lex.TokenizeAll(e.String())
	return tokens, diags
}

func (e *ExportRecord) SetValue(s string) {
	e.Value = s
}

type ExportRecords []*ExportRecord

func (e ExportRecords) String() string {
	s := "{\n"
	for _, r := range e {
		s += r.String()
	}
	s += "}\n"
	return s
}

func (e ExportRecords) Tokens() (hclwrite.Tokens, hcl.Diagnostics) {
	tokens := hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenOBrace,
			Bytes: []byte("{"),
		},
		&hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		},
	}

	diags := hcl.Diagnostics{}
	for _, r := range e {
		t, d := r.Tokens()
		diags = append(diags, d...)
		tokens = append(tokens, t...)
	}

	tokens = append(tokens, hclwrite.Tokens{
		&hclwrite.Token{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte("}"),
		},
		//&hclwrite.Token{
		//	Type:  hclsyntax.TokenNewline,
		//	Bytes: []byte("\n"),
		//},
	}...)

	return tokens, diags
}

// Parse a struct (tokenized by hclwrite) like:
//
//	exports = {
//	  DB_PORT = "3306"
//	  DB_PASSWORD = decrypt("DEV", "c3VwZXJTZWNyZXRQcm9k")
//	}
//
// into a list of ExportRecord
func Parse(exportsTokens hclwrite.Tokens) (ExportRecords, hcl.Diagnostics) {
	t := newTree()
	t.srcTokens = exportsTokens
	exportRecords, diags := t.parse()
	return exportRecords, diags
}

type Tree struct {
	// parsing only, cleared after parse
	index     int             // where we are in the token list
	srcTokens hclwrite.Tokens // tokens from hclwrite lexer
}

func newTree() *Tree {
	return &Tree{}
}

// parse exports
//
// Duplicates are allowed here.  You have handle duplicates at a higher level.
// The order is preserved, so you can decide to take the first or last.
//
// we've got something like
//
//		exports = {
//	      #DB_USER = "db"
//		  DB_PORT = "3306"
//		  DB_PASSWORD = decrypt("DEV", "c3VwZXJTZWNyZXRQcm9k")
//		}
//
// which would have tokens:
//
//		                      S=SpacesBefore
//		                      |
//		 {                    1 (TokenOBrace)
//		 \n                   0 (TokenNewline)
//		 #DB_USER = "db"      4 (TokenComment)    - this may or may not be here, notice there is no TokenNewline after TokenComment
//		 DB_PORT              4 (TokenIdent)
//		 =                    1 (TokenEqual)
//		 "                    0 (TokenOQuote)
//		 3306                 0 (TokenQuotedLit)
//		 "                    0 (TokenCQuote)
//		 \n                   0 (TokenNewline)
//		 DB_PASSWORD          4 (TokenIdent)
//		 =                    1 (TokenEqual)
//		 decrypt              1 (TokenIdent)
//		 (                    0 (TokenOParen)
//	     "                    0 (TokenOQuote)
//		 DEV                  0 (TokenQuotedLit)
//		 "                    0 (TokenCQuote)
//		 ,                    0 (TokenComma)
//		 "                    1 (TokenOQuote)
//		 c3VwZXJTZWNyZXRQcm9k 0 (TokenQuotedLit)
//		 "                    0 (TokenCQuote)
//		 )                    0 (TokenCParen)
//		 \n                   0 (TokenNewline)
//		 }                    2 (TokenCBrace)
//
// NOTE: Comments are not parsed, so we store them in .Name
//
//	You can detect this by looking for .Value being empty
//	In the example above, we would end up with
//	.Name = `    #DB_USER = "db"`
//	When iterating through the list of exports,
//	check if .Value == "", and if so, you do not need
//	to reconstruct the line with Sprintf("%s = %s", .Name, .Value)
//	Instead, you would just do Sprintf("%s", .Name)
func (t *Tree) parse() (ExportRecords, hcl.Diagnostics) {
	// TODO: add diags
	var exportRecords ExportRecords
	eqlFound := false
	key := ""
	val := ""
	atValEnd := false

	reset := func() {
		eqlFound = false
		key = ""
		val = ""
		atValEnd = false
	}

	// TODO(rchapman): parser could probably use some cleanup.
	//                 Not a big deal for now since it's only used by
	//                 the add_secret program.
	for {
		token := t.next()
		if token == nil {
			break
		}

		// comments are not terminated with TokenNewline (it's implied)
		if token.Type == hclsyntax.TokenComment {
			for i := 0; i < token.SpacesBefore; i++ {
				key += " "
			}
			key += string(token.Bytes)
			atValEnd = true
		} else {
			// on newline, reset eqlFound, key and val
			if token.Type == hclsyntax.TokenNewline {
				reset()
				continue
			}

			// look for `=`
			if token.Type == hclsyntax.TokenEqual {
				eqlFound = true
				continue
			}

			// everything after newline/comment and before `=` is the key (.Name)
			if !eqlFound {
				for i := 0; i < token.SpacesBefore; i++ {
					key += " "
				}
				key += string(token.Bytes)
				continue
			}

			// e.g. `DB_PASSWORD =` already found
			if eqlFound {
				// build up val (RHS of `=`)
				for i := 0; i < token.SpacesBefore; i++ {
					val += " "
				}
				val += string(token.Bytes)

				// are we at the end of val tokens?
				peekToken := t.peek()

				if peekToken == nil || peekToken.Type == hclsyntax.TokenNewline {
					atValEnd = true
				}

			}
		}

		if atValEnd {
			record := ExportRecord{
				Name:  key,
				Value: val,
			}
			exportRecords = append(exportRecords, &record)
			// on append, reset eqlFound, key and val
			reset()
		}
	}

	return exportRecords, nil
}

func (t *Tree) peek() *hclwrite.Token {
	i := t.index
	next := i + 1

	if next > len(t.srcTokens) {
		return nil
	}
	return t.srcTokens[next]
}

// return next hclwrite.Token, or nil if at end of token list
func (t *Tree) next() *hclwrite.Token {
	next := t.index + 1

	if next >= len(t.srcTokens) {
		return nil
	}

	t.index = next
	return t.srcTokens[next]
}
