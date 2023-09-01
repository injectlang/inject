package lex

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// TokenizeAll tokenizes a string, like
//
//	first_name = "tim"
//
//	 or
//
//	DB_PASSWORD = decrypt("DEV", "c3VwZXJTZWNyZXRQcm9k")
//
// returns all the tokens in the string
// (including LHS, `=`, and RHS)
//
// for example, calling
//
//	TokenizeAll(`DB_PASSWORD = decrypt("DEV", "c3VwZXJTZWNyZXRQcm9k")`)
//
// would return the tokenization for
//
//	DB_PASSWORD = decrypt("DEV", "c3VwZXJTZWNyZXRQcm9k")
func TokenizeAll(s string) (hclwrite.Tokens, hcl.Diagnostics) {
	actionTxt := fmt.Sprintf("TokenizeAll")
	f, diags := hclwrite.ParseConfig([]byte(s), actionTxt, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}

	tokens := f.Body().BuildTokens(nil)
	return tokens, diags
}

// TokenizeAttrRHS tokenizes a string with a single attribute, like
//
//		first_name = "tim"
//
//	 or
//
//		DB_PASSWORD = decrypt("DEV", "c3VwZXJTZWNyZXRQcm9k")
//
// returns tokens on right-hand-side (RHS), after `=` (TokenEquals)
// returns nil if the attr cannot be found in the string
//
// for example, calling
//
//	TokenizeAttrRHS(`DB_PASSWORD = decrypt("DEV", "c3VwZXJTZWNyZXRQcm9k")`, "DB_PASSWORD")
//
// would return the tokenization for
//
//	decrypt("DEV", "c3VwZXJTZWNyZXRQcm9k")
func TokenizeAttrRHS(s, attrName string) (hclwrite.Tokens, hcl.Diagnostics) {
	actionTxt := fmt.Sprintf("TokenizeRHS_%s", attrName)
	f, diags := hclwrite.ParseConfig([]byte(s), actionTxt, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags
	}
	attr := f.Body().GetAttribute(attrName)
	if attr == nil {
		return nil, diags
	}

	tokens := attr.Expr().BuildTokens(nil)
	return tokens, diags
}
