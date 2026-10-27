package hub_test

import (
	"os"
	"testing"
)

// TestMain: no test makes a keychain for the hub's agy home.
func TestMain(m *testing.M) {
	os.Setenv("OMNILLM_AGY_KEYCHAIN", "off")
	os.Exit(m.Run())
}
