package injector

import (
	"fmt"
	"os"
	"testing"
)

func TestNewConfigFile_Parse(t *testing.T) {
	input := []byte(`// dev context
context "dev" {
  exports = {
    DB_USER = "user"
    DB_PASSWORD = "pass"
  }
}

context "prod" {
  exports = {
    DB_USER = "user"
    DB_PASSWORD = "pass"
  }
}
`)

	t.Run(fmt.Sprintf("%02d", 1), func(t *testing.T) {
		filename := fmt.Sprintf("%s/test_input.hcl", t.TempDir())
		err := os.WriteFile(filename, input, 0644)
		if err != nil {
			t.Fatalf("os.WriteFile(\"%s\", ...)", filename)
		}
		cf := NewConfigFile(filename)

		verifyExports := func(exports map[string]string) bool {
			if exports["DB_USER"] != "user" || exports["DB_PASSWORD"] != "pass" {
				return false
			}
			return true
		}

		devOK := false
		prodOK := false
		for _, c := range cf.Contexts {
			if c.Name == "dev" {
				if !verifyExports(c.Exports) {
					t.Fatalf("could not verify exports in context \"%s\"", c.Name)
				}
				devOK = true
			}
			if c.Name == "prod" {
				if !verifyExports(c.Exports) {
					t.Fatalf("could not verify exports in context \"%s\"", c.Name)
				}
				prodOK = true
			}
		}

		if !devOK {
			t.Fatalf("could not find a context named \"dev\"")
		}
		if !prodOK {
			t.Fatalf("could not find a context name \"prod\"")
		}
	})
}
