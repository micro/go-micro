package model_test

import (
	"go-micro.dev/v6/model"
	_ "go-micro.dev/v6/model/mistral"
	_ "go-micro.dev/v6/model/openai"
	"testing"
)

func TestProviderSelection(t *testing.T) {
	if m := model.New(""); m == nil || m.String() != "openai" {
		t.Fatal("empty provider must select OpenAI")
	}
	if model.New("misspelled-provider") != nil {
		t.Fatal("unknown provider must not silently select another provider")
	}
	if m := model.New("mistral"); m == nil || m.String() != "mistral" {
		t.Fatal("explicit provider must be preserved")
	}
}
