package editfile

import (
	"fmt"
	"os"
	"testing"
)

// TODO: does hclwrite handle duplicate blocks?
//       test context, public_key, custom_function

func TestNewEditConfigFile_AddPublicKey(t *testing.T) {
	tests := []struct {
		input      string
		pubkey     string
		pubkeyName string
		want       string
	}{
		{
			``,
			`{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePublicKey","value":"EgYIARABGAIaIJO9jY9uaVIcwcl4EuKYjRaCJIsVKdGoC6akdYTP1VxK","keyMaterialType":"ASYMMETRIC_PUBLIC"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}`,
			"A",
			`
public_key "A" {
  base64 = <<-EOT
    eyJwcmltYXJ5S2V5SWQiOjYxMzg0NDUsImtleSI6W3sia2V5RGF0YSI6eyJ0eXBl
    VXJsIjoidHlwZS5nb29nbGVhcGlzLmNvbS9nb29nbGUuY3J5cHRvLnRpbmsuSHBr
    ZVB1YmxpY0tleSIsInZhbHVlIjoiRWdZSUFSQUJHQUlhSUpPOWpZOXVhVkljd2Ns
    NEV1S1lqUmFDSklzVktkR29DNmFrZFlUUDFWeEsiLCJrZXlNYXRlcmlhbFR5cGUi
    OiJBU1lNTUVUUklDX1BVQkxJQyJ9LCJzdGF0dXMiOiJFTkFCTEVEIiwia2V5SWQi
    OjYxMzg0NDUsIm91dHB1dFByZWZpeFR5cGUiOiJUSU5LIn1dfQ==
  EOT
}


`,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			input := []byte(test.input)
			filename := fmt.Sprintf("%s/test_input.hcl", t.TempDir())
			err := os.WriteFile(filename, input, 0644)
			if err != nil {
				t.Fatalf("os.WriteFile(\"%s\", ...)", filename)
			}
			edit := NewEditConfigFile(filename)
			edit.AddPublicKey(test.pubkeyName, []byte(test.pubkey), false)
			src, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("os.ReadFile(\"%s\", ...)", filename)
			}

			got := string(src)
			if got != test.want {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func TestNewEditConfigFile_AddPublicKey_Name(t *testing.T) {
	tests := []struct {
		pubkeyName string
		want       bool // valid name?
	}{
		{
			"DEV2022",
			true,
		},
		{
			"Dev2022",
			false,
		},
		{
			"a",
			false,
		},
		{
			"De",
			false,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			edit := NewEditConfigFile("blah")
			got := edit.validatePubkeyName(test.pubkeyName)

			if got != test.want {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func TestEditConfigFile_Sort(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			`// context z
context "z" {}

// public_key z
public_key "z" {}

// context a
context "a" {}

// custom_function z
custom_function "z" {}

// public_key a
public_key "a" {}

// custom_function a
custom_function "a" {}
`,
			`// custom_function a
custom_function "a" {}

// custom_function z
custom_function "z" {}

// public_key a
public_key "a" {}

// public_key z
public_key "z" {}

// context a
context "a" {}

// context z
context "z" {}

`,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			input := []byte(test.input)
			filename := fmt.Sprintf("%s/test_input.hcl", t.TempDir())
			err := os.WriteFile(filename, input, 0644)
			if err != nil {
				t.Fatalf("os.WriteFile(\"%s\", ...)", filename)
			}
			edit := NewEditConfigFile(filename)
			diags := edit.parse()
			if diags.HasErrors() {
				t.Fatalf("parse(): %#v", diags)
			}
			got := string(sortBlocks(edit.file).Bytes())
			if got != test.want {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}
