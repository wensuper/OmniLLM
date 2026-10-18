// Package secret is where credentials and private keys live (principle 3).
// The host passes a Store; the standalone program uses the system keychain
// or, on a machine without one, a file encrypted with a master key.
package secret

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/crypto/chacha20poly1305"
)

// ErrNotFound is returned for a missing name.
var ErrNotFound = errors.New("secret: not found")

// Store holds named secrets. Names are ASCII paths like "hub/static_key".
type Store interface {
	Get(ctx context.Context, name string) ([]byte, error)
	Set(ctx context.Context, name string, value []byte) error
	Delete(ctx context.Context, name string) error
	// List returns names under prefix (never values).
	List(ctx context.Context, prefix string) ([]string, error)
}

// ValidName rejects names that could escape a namespace.
func ValidName(name string) error {
	if name == "" || len(name) > 200 || strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
		return errors.New("secret: bad name")
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("/_-.", r)) {
			return errors.New("secret: bad name")
		}
	}
	return nil
}

// Memory is an in-process store for tests and short-lived hosts.
type Memory struct {
	mu sync.Mutex
	m  map[string][]byte
}

func NewMemory() *Memory { return &Memory{m: map[string][]byte{}} }

func (s *Memory) Get(_ context.Context, name string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[name]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte{}, v...), nil
}

func (s *Memory) Set(_ context.Context, name string, v []byte) error {
	if err := ValidName(name); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[name] = append([]byte{}, v...)
	return nil
}

func (s *Memory) Delete(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, name)
	return nil
}

func (s *Memory) List(_ context.Context, prefix string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for k := range s.m {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out, nil
}

// File stores all secrets in one file sealed with ChaCha20-Poly1305 under a
// 32-byte master key the host supplies (environment variable, systemd
// credential, KMS). The file is written 0600 via an atomic rename.
type File struct {
	path string
	aead interface {
		Seal(dst, nonce, plaintext, additionalData []byte) []byte
		Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error)
		NonceSize() int
	}
	mu sync.Mutex
}

// NewFile opens (or will create) an encrypted secret file.
func NewFile(path string, masterKey []byte) (*File, error) {
	if len(masterKey) != chacha20poly1305.KeySize {
		return nil, errors.New("secret: master key must be 32 bytes")
	}
	a, err := chacha20poly1305.NewX(masterKey)
	if err != nil {
		return nil, err
	}
	return &File{path: path, aead: a}, nil
}

// ParseMasterKey decodes a base64 master key (as kept in an env var).
func ParseMasterKey(s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil || len(b) != 32 {
		return nil, errors.New("secret: master key must be 32 bytes, base64")
	}
	return b, nil
}

var fileAD = []byte("omnillm/secret-file/1")

func (f *File) load() (map[string][]byte, error) {
	b, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string][]byte{}, nil
	}
	if err != nil {
		return nil, err
	}
	ns := f.aead.NonceSize()
	if len(b) < ns {
		return nil, errors.New("secret: file corrupt")
	}
	plain, err := f.aead.Open(nil, b[:ns], b[ns:], fileAD)
	if err != nil {
		return nil, errors.New("secret: wrong master key or file tampered")
	}
	m := map[string][]byte{}
	if err := json.Unmarshal(plain, &m); err != nil {
		return nil, errors.New("secret: file corrupt")
	}
	return m, nil
}

func (f *File) save(m map[string][]byte) error {
	plain, err := json.Marshal(m)
	if err != nil {
		return err
	}
	nonce := make([]byte, f.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	return WriteFileAtomic(f.path, append(nonce, f.aead.Seal(nil, nonce, plain, fileAD)...))
}

func (f *File) Get(_ context.Context, name string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.load()
	if err != nil {
		return nil, err
	}
	v, ok := m[name]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}

func (f *File) Set(_ context.Context, name string, v []byte) error {
	if err := ValidName(name); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.load()
	if err != nil {
		return err
	}
	m[name] = append([]byte{}, v...)
	return f.save(m)
}

func (f *File) Delete(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.load()
	if err != nil {
		return err
	}
	delete(m, name)
	return f.save(m)
}

func (f *File) List(_ context.Context, prefix string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.load()
	if err != nil {
		return nil, err
	}
	var out []string
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out, nil
}

// WriteFileAtomic writes b to path with mode 0600 via a temp file and rename,
// creating the parent directory 0700.
func WriteFileAtomic(path string, b []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("secret: rename: %w", err)
	}
	return nil
}
