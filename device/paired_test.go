package device

import (
	"context"
	"errors"
	"testing"

	"github.com/wensuper/OmniLLM/secret"
)

type brokenStore struct{ secret.Store }

func (brokenStore) Get(context.Context, string) ([]byte, error) {
	return nil, errors.New("keyring locked")
}

// A pairing that can't be read is not "not paired": pairing over it would
// throw away a working pairing.
func TestCheckPaired(t *testing.T) {
	ctx := context.Background()
	m := secret.NewMemory()
	if ok, err := CheckPaired(ctx, m); ok || err != nil {
		t.Fatalf("empty: %v %v", ok, err)
	}
	if err := m.Set(ctx, sID, []byte("d1")); err != nil {
		t.Fatal(err)
	}
	if ok, err := CheckPaired(ctx, m); !ok || err != nil {
		t.Fatalf("paired: %v %v", ok, err)
	}
	if ok, err := CheckPaired(ctx, brokenStore{}); ok || err == nil {
		t.Fatalf("unreadable: %v %v", ok, err)
	}
	if Paired(ctx, brokenStore{}) {
		t.Fatal("Paired on an unreadable store")
	}
}
