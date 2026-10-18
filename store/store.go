// Package store is the library's non-secret state: devices (public keys
// only), budgets, usage and audit. Hosts may pass their own implementation
// (for NEXA, a table in its own database); the default is one JSON file
// written 0600 with an atomic rename. Nothing secret ever goes here.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/wensuper/OmniLLM/secret"
)

// ErrNotFound is returned for a missing key.
var ErrNotFound = errors.New("store: not found")

// Store is a small key-value store.
type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
}

// Memory is an in-process store.
type Memory struct {
	mu sync.Mutex
	m  map[string][]byte
}

func NewMemory() *Memory { return &Memory{m: map[string][]byte{}} }

func (s *Memory) Get(_ context.Context, k string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[k]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte{}, v...), nil
}

func (s *Memory) Put(_ context.Context, k string, v []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[k] = append([]byte{}, v...)
	return nil
}

func (s *Memory) Delete(_ context.Context, k string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, k)
	return nil
}

func (s *Memory) List(_ context.Context, prefix string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return listKeys(s.m, prefix), nil
}

func listKeys(m map[string][]byte, prefix string) []string {
	var out []string
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// File keeps everything in one JSON file (fine for one person's devices).
type File struct {
	path string
	mu   sync.Mutex
	m    map[string][]byte
}

// OpenFile loads path, or starts empty if it does not exist.
func OpenFile(path string) (*File, error) {
	f := &File{path: path, m: map[string][]byte{}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &f.m); err != nil {
		return nil, errors.New("store: file corrupt")
	}
	return f, nil
}

func (f *File) flush() error {
	b, err := json.Marshal(f.m)
	if err != nil {
		return err
	}
	return secret.WriteFileAtomic(f.path, b)
}

func (f *File) Get(_ context.Context, k string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.m[k]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte{}, v...), nil
}

func (f *File) Put(_ context.Context, k string, v []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[k] = append([]byte{}, v...)
	return f.flush()
}

func (f *File) Delete(_ context.Context, k string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.m, k)
	return f.flush()
}

func (f *File) List(_ context.Context, prefix string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return listKeys(f.m, prefix), nil
}

// GetJSON / PutJSON are helpers.
func GetJSON(ctx context.Context, s Store, k string, v any) error {
	b, err := s.Get(ctx, k)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func PutJSON(ctx context.Context, s Store, k string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.Put(ctx, k, b)
}
