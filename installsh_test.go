package omnillm

import (
	"os"
	"regexp"
	"testing"
)

// In a Chinese locale sh reads a non-ASCII character right after $NAME as
// part of the name ("$REPO（" is the variable "REPO（"), so install.sh must
// brace every variable that is followed by one.
func TestInstallShBracesVariablesBeforeNonASCII(t *testing.T) {
	b, err := os.ReadFile("install.sh")
	if err != nil {
		t.Fatal(err)
	}
	if m := regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_]*[^\x00-\x7F]`).FindAll(b, -1); len(m) > 0 {
		t.Fatalf("unbraced variables before non-ASCII text in install.sh: %q", m)
	}
}
