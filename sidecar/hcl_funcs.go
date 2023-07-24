package sidecar

//
// HCL functions includes functions written in golang that can be invoked from
// hcl config files as well as custom functions. Custom functions are defined
// in the hcl config file.
//

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-cty-funcs/cidr"
	"github.com/hashicorp/go-cty-funcs/crypto"
	"github.com/hashicorp/go-cty-funcs/encoding"
	"github.com/hashicorp/go-cty-funcs/uuid"
	"github.com/hashicorp/hcl/v2/ext/tryfunc"
	"github.com/hashicorp/hcl/v2/ext/typeexpr"
	ctyyaml "github.com/zclconf/go-cty-yaml"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

type HclFuncs struct {
	funcMap map[string]function.Function
}

// Custom Functions refers to any functions defined in the hcl file
// by a user
func NewHclFuncs(customFuncMap map[string]function.Function) *HclFuncs {
	hf := HclFuncs{}

	builtinFuncs := map[string]function.Function{
		"abs":          stdlib.AbsoluteFunc,
		"base64decode": encoding.Base64DecodeFunc,
		"base64encode": encoding.Base64EncodeFunc,
		"bcrypt":       crypto.BcryptFunc,
		"can":          tryfunc.CanFunc,
		"ceil":         stdlib.CeilFunc,
		"chomp":        stdlib.ChompFunc,
		"chunklist":    stdlib.ChunklistFunc,
		"cidrhost":     cidr.HostFunc,
		"cidrnetmask":  cidr.NetmaskFunc,
		"cidrsubnet":   cidr.SubnetFunc,
		"cidrsubnets":  cidr.SubnetsFunc,
		"coalesce":     stdlib.CoalesceFunc,
		"coalescelist": stdlib.CoalesceListFunc,
		"compact":      stdlib.CompactFunc,
		"concat":       stdlib.ConcatFunc,
		"contains":     stdlib.ContainsFunc,
		"convert":      typeexpr.ConvertFunc,
		"csvdecode":    stdlib.CSVDecodeFunc,
		"decrypt":      DecryptFunc,
		"distinct":     stdlib.DistinctFunc,
		"element":      stdlib.ElementFunc,
		"flatten":      stdlib.FlattenFunc,
		"floor":        stdlib.FloorFunc,
		"format":       stdlib.FormatFunc,
		"formatdate":   stdlib.FormatDateFunc,
		"formatlist":   stdlib.FormatListFunc,
		"indent":       stdlib.IndentFunc,
		"index":        stdlib.IndexFunc,
		//"isdigits":      isDigitsFunc,
		//"isemail":       isEmailFunc,
		//"isupper":       isUpperFunc,
		//"islower":       isLowerFunc,
		"join":            stdlib.JoinFunc,
		"jsondecode":      stdlib.JSONDecodeFunc,
		"jsonencode":      stdlib.JSONEncodeFunc,
		"keys":            stdlib.KeysFunc,
		"length":          stdlib.LengthFunc,
		"log":             stdlib.LogFunc,
		"lookup":          stdlib.LookupFunc,
		"lower":           stdlib.LowerFunc,
		"max":             stdlib.MaxFunc,
		"md5":             crypto.Md5Func,
		"merge":           stdlib.MergeFunc,
		"min":             stdlib.MinFunc,
		"parseint":        stdlib.ParseIntFunc,
		"pow":             stdlib.PowFunc,
		"range":           stdlib.RangeFunc,
		"reverse":         stdlib.ReverseFunc,
		"replace":         stdlib.ReplaceFunc,
		"regex_replace":   stdlib.RegexReplaceFunc,
		"rsadecrypt":      crypto.RsaDecryptFunc,
		"setintersection": stdlib.SetIntersectionFunc,
		"setproduct":      stdlib.SetProductFunc,
		"setunion":        stdlib.SetUnionFunc,
		"sha1":            crypto.Sha1Func,
		"sha256":          crypto.Sha256Func,
		"sha512":          crypto.Sha512Func,
		"signum":          stdlib.SignumFunc,
		"slice":           stdlib.SliceFunc,
		"sort":            stdlib.SortFunc,
		"split":           stdlib.SplitFunc,
		"strlen":          stdlib.StrlenFunc,
		"strrev":          stdlib.ReverseFunc,
		"substr":          stdlib.SubstrFunc,
		"timeadd":         stdlib.TimeAddFunc,
		"title":           stdlib.TitleFunc,
		"trim":            stdlib.TrimFunc,
		"trimprefix":      stdlib.TrimPrefixFunc,
		"trimspace":       stdlib.TrimSpaceFunc,
		"trimsuffix":      stdlib.TrimSuffixFunc,
		"try":             tryfunc.TryFunc,
		"upper":           stdlib.UpperFunc,
		"urlencode":       encoding.URLEncodeFunc,
		"uuidv4":          uuid.V4Func,
		"uuidv5":          uuid.V5Func,
		"values":          stdlib.ValuesFunc,
		"yamldecode":      ctyyaml.YAMLDecodeFunc,
		"yamlencode":      ctyyaml.YAMLEncodeFunc,
		"zipmap":          stdlib.ZipmapFunc,
	}

	// add csutom funcs first to allFuncMap. Even though we are checking for
	// custom functions attempting to override built-in funcs above, adding
	// built-in funcs last ensures they will take precedence if we had a bug.
	allFuncMap := customFuncMap
	for name, ctyFunc := range customFuncMap {
		if _, found := builtinFuncs[name]; found {
			log.Fatalf("custom function named \"%s\" cannot override built-in function", name)
		}
		allFuncMap[name] = ctyFunc
	}
	for name, ctyFunc := range builtinFuncs {
		allFuncMap[name] = ctyFunc
	}

	hf.funcMap = allFuncMap

	return &hf
}

func (hf *HclFuncs) FuncMap() map[string]function.Function {
	return hf.funcMap
}

var DecryptFunc = function.New(&function.Spec{
	Description: "Decrypt a base64 encoded string using a asymmetric private key and HPKE.",
	Params: []function.Parameter{
		{
			Name:        "keypairName",
			Description: "The name of the keypair used to encrypt the plaintext.",
			Type:        cty.String,
		},
		{
			Name:        "encryptedBase64Str",
			Description: "The base64 encoded encrypted string to decrypt.",
			Type:        cty.String,
		},
	},
	Type:         function.StaticReturnType(cty.String),
	RefineResult: refineNonNull,
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		keypairName := args[0].AsString()
		encryptedBase64Str := args[1].AsString()
		privateEnvName := fmt.Sprintf("PRIVATE_JSON_KEYSET_%s", keypairName)
		privateJSONKeysetB64 := os.Getenv(privateEnvName)
		if privateJSONKeysetB64 == "" {
			err2 := fmt.Errorf("Env var %s must be set", privateEnvName)
			return cty.NilVal, err2
		}
		privateJSONKeysetBytes, err := base64.StdEncoding.DecodeString(privateJSONKeysetB64)
		if err != nil {
			err2 := fmt.Errorf("could not base64 decode string in env var \"%s\": %+v", privateEnvName, err)
			return cty.NilVal, err2
		}
		privateJSONKeyset := string(privateJSONKeysetBytes)
		decryptor := NewDecryptor(privateJSONKeyset)
		ciphertext, err := base64.StdEncoding.DecodeString(encryptedBase64Str)
		if err != nil {
			err2 := fmt.Errorf("could not base64 decode ciphertext \"%s\": %+v", encryptedBase64Str, err)
			return cty.NilVal, err2
		}
		encryptionContext := []byte(nil)
		plaintextBytes, err := decryptor.Decrypt([]byte(ciphertext), encryptionContext)
		if err != nil {
			err2 := fmt.Errorf("could not decrypt: %+v", err)
			return cty.NilVal, err2
		}
		plaintext := string(plaintextBytes)
		return cty.StringVal(plaintext), nil
	},
})

func refineNonNull(b *cty.RefinementBuilder) *cty.RefinementBuilder {
	return b.NotNull()
}
