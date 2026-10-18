package secret

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/zalando/go-keyring"
)

// Keyring keeps secrets in the system keychain (macOS Keychain, Windows
// Credential Manager, Secret Service on Linux). The keychain has no listing
// call, so an index of names (never values) is kept as one more entry.
type Keyring struct {
	service string
	mu      sync.Mutex
}

// NewKeyring uses service as the keychain service name, e.g. "omnillm".
func NewKeyring(service string) *Keyring { return &Keyring{service: service} }

const indexName = "_index"

func (k *Keyring) Get(_ context.Context, name string) ([]byte, error) {
	v, err := keyring.Get(k.service, name)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(v)
}

func (k *Keyring) Set(ctx context.Context, name string, v []byte) error {
	if err := ValidName(name); err != nil {
		return err
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := keyring.Set(k.service, name, base64.StdEncoding.EncodeToString(v)); err != nil {
		return err
	}
	return k.updateIndex(func(m map[string]bool) { m[name] = true })
}

func (k *Keyring) Delete(_ context.Context, name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := keyring.Delete(k.service, name); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return k.updateIndex(func(m map[string]bool) { delete(m, name) })
}

func (k *Keyring) List(_ context.Context, prefix string) ([]string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	m, err := k.index()
	if err != nil {
		return nil, err
	}
	var out []string
	for n := range m {
		if strings.HasPrefix(n, prefix) {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (k *Keyring) index() (map[string]bool, error) {
	m := map[string]bool{}
	v, err := keyring.Get(k.service, indexName)
	if errors.Is(err, keyring.ErrNotFound) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	if err := json.Unmarshal([]byte(v), &names); err != nil {
		return nil, errors.New("secret: keychain index is corrupt")
	}
	for _, n := range names {
		m[n] = true
	}
	return m, nil
}

func (k *Keyring) updateIndex(f func(map[string]bool)) error {
	m, err := k.index()
	if err != nil {
		return err
	}
	f(m)
	names := make([]string, 0, len(m))
	for n := range m {
		names = append(names, n)
	}
	sort.Strings(names)
	b, _ := json.Marshal(names)
	return keyring.Set(k.service, indexName, string(b))
}
