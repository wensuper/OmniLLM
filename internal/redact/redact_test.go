package redact

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	cases := []string{
		"Authorization: Bearer sk-abcdefghijklmnop",
		"x-api-key=sk-ant-api03-ABCDEFGHIJKLMNOP",
		`{"api_key":"abcd1234efgh5678"}`,
		"token omni_app_0123456789abcdef0123",
		"cookie: session=abcdef0123456789",
		"jwt eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		"omnillm1-eyJyIjoid3NzOi8vcmVsYXkuZXhhbXBsZS5jb20iLCJtIjoiYWJhYmFi",
	}
	for _, c := range cases {
		out := String(c)
		for _, secret := range []string{"abcdefghijklmnop", "ABCDEFGHIJKLMNOP", "abcd1234efgh5678", "0123456789abcdef0123", "abcdef0123456789", "SflKxwRJSMeKKF2QT4fw", "eyJyIjoid3NzOi8vcmVsYXkuZXhhbXBsZS5jb20iLCJtIjoiYWJhYmFi"} {
			if strings.Contains(out, secret) {
				t.Fatalf("%q → %q still holds %q", c, out, secret)
			}
		}
	}
	if String("device paired, 3 models") != "device paired, 3 models" {
		t.Fatal("ordinary text changed")
	}
}

func TestLast4(t *testing.T) {
	if Last4("sk-0123456789abcdefghijwxyz") != "wxyz" || Last4("sk-0123456789abcd") != "????" || Last4("short") != "????" {
		t.Fatal(Last4("sk-0123456789abcdefghijwxyz"), Last4("sk-0123456789abcd"))
	}
}
