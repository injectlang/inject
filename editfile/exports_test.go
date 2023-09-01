package editfile

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"testing"
)

func TestValidExportName(t *testing.T) {
	tests := []struct {
		exportName string
		want       bool
	}{
		{
			"DB_USER1",
			true,
		},
		{
			"db_user1",
			false,
		},
		{
			"dB_USER1",
			false,
		},
		{
			"Db_user1",
			false,
		},
		{
			"DB_USEr",
			false,
		},
		{
			"Db_User",
			false,
		},
		{
			"1DB",
			false,
		},
		{
			"_DB",
			true,
		},
		{
			"_1DB",
			true,
		},
		{
			"_1db",
			false,
		},
		{
			"_1",
			true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			got := validateName(test.exportName)
			if got != test.want {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func TestDuplicates(t *testing.T) {
	checkDiags := func(diags hcl.Diagnostics, errmsg string) {
		if diags.HasErrors() {
			t.Errorf(errmsg)
			for _, diag := range diags {
				t.Logf("- %s", diag)
			}
		}
	}

	src := `
context "dev" {
  exports = {
    DB_NAME = "app1"
    DB_USER = "db"
    DB_PASSWORD = decrypt("DEV", "c3VwZXJTZWNyZXRQcm9k")
  }
}`
	file, diags := hclwrite.ParseConfig([]byte(src), "test", hcl.InitialPos)
	checkDiags(diags, "could not parse hclwrite.ParseConfig(src, ...)")

	exports := NewExports(file, "dev")

	exists, diags := exports.Exists("DB_PASSWORD")
	checkDiags(diags, "could not call exports.Exists(\"DB_PASSWORD\")")
	if !exists {
		t.Errorf("exports.Exists() did not detect that an export already exists\ngot:  Exists()=false, want: Exists()=true\n")
	}

	diags = exports.SetEncryptedValue("DB_PASSWORD", "KP1", "s3cr3t", false)
	got := diags.HasErrors()
	want := true
	if got != want {
		t.Errorf("did not detect duplicates\ngot:  diags.HasErrors()=%#v\nwant: diags.HasErrors()=%#v", got, want)
	}

}
