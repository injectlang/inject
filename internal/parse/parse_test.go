package parse

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		src   string
		mutFn func(records ExportRecords)
		want  string
	}{
		{`exports = {
    #DB_USER = "app1"
    DB_NAME = "db"
    DB_PW = decrypt("KEY1", "abcde")
}`,
			func(records ExportRecords) {
				records[1].SetValue(` "db2"`)
			},
			`{
    #DB_USER = "app1"
    DB_NAME = "db2"
    DB_PW = decrypt("KEY1", "abcde")
}
`,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%02d", i), func(t *testing.T) {
			checkDiags := func(diags hcl.Diagnostics, errmsg string) {
				if diags.HasErrors() {
					t.Errorf(errmsg)
					for _, diag := range diags {
						t.Logf("- %s", diag)
					}
				}
			}

			f, diags := hclwrite.ParseConfig([]byte(test.src), "test", hcl.InitialPos)
			checkDiags(diags, "could not parse hclwrite.ParseConfig(src, ...)")

			exportsAttr := f.Body().GetAttribute("exports")
			tokens := exportsAttr.Expr().BuildTokens(nil)

			exportRecords, diags := Parse(tokens)
			test.mutFn(exportRecords)
			got := exportRecords.String()

			if got != test.want {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}
