package animate

import (
	"testing"
)

func TestAnimate(t *testing.T) {
	// TODO: fix test
	return

	testData := []struct {
		text string
	}{
		{"funny cat"},
	}

	command := Animate()

	for _, d := range testData {
		rsp, err := command.Exec("animate", d.text)
		if err != nil {
			t.Fatal(err)
		}

		if rsp == nil {
			t.Fatal("expected result, got nil")
		}
	}
}
