package hub

import (
	"reflect"
	"testing"
)

func TestChatModels(t *testing.T) {
	in := []string{"bge-m3:latest", "qwen3:8b", "nomic-embed-text:latest", "llama3.2", "mxbai-embed-large", "bge-reranker-v2-m3", "all-minilm", "gemma3:4b"}
	want := []string{"qwen3:8b", "llama3.2", "gemma3:4b"}
	if got := ChatModels(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	if ChatModels([]string{"bge-m3:latest"}) != nil {
		t.Fatal("an embedding-only server must yield no chat models")
	}
}
