package secret

import (
	"context"
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestKeyringWithMock(t *testing.T) {
	keyring.MockInit() // never touches the real keychain
	k := NewKeyring("omnillm-test")
	ctx := context.Background()
	if err := k.Set(ctx, "provider/a/key", []byte("v1")); err != nil {
		t.Fatal(err)
	}
	_ = k.Set(ctx, "hub/static_key", []byte{0, 1, 2, 255})
	v, err := k.Get(ctx, "hub/static_key")
	if err != nil || string(v) != string([]byte{0, 1, 2, 255}) {
		t.Fatal("binary values must round-trip", v, err)
	}
	names, _ := k.List(ctx, "provider/")
	if len(names) != 1 || names[0] != "provider/a/key" {
		t.Fatal(names)
	}
	_ = k.Delete(ctx, "provider/a/key")
	if _, err := k.Get(ctx, "provider/a/key"); !errors.Is(err, ErrNotFound) {
		t.Fatal("delete")
	}
	if names, _ := k.List(ctx, "provider/"); len(names) != 0 {
		t.Fatal("index not updated", names)
	}
	if err := k.Set(ctx, "../x", nil); err == nil {
		t.Fatal("bad name accepted")
	}
	// A corrupt index is an error, not a silent empty list.
	_ = keyring.Set("omnillm-test", indexName, "{not json")
	if _, err := k.List(ctx, ""); err == nil {
		t.Fatal("corrupt index accepted")
	}
}
