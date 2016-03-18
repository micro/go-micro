package errors

import (
	"net/http"
	"testing"
)

func TestErrors(t *testing.T) {
	testData := []*Error{
		&Error{
			Id:     "test",
			Code:   500,
			Detail: "Internal server error",
			Status: http.StatusText(500),
		},
	}

	for _, e := range testData {
		ne := New(e.Id, e.Detail, e.Code)

		if e.Error() != ne.Error() {
			t.Fatal("Expected %s got %s", e.Error(), ne.Error())
		}

		pe := Parse(ne.Error())

		if pe == nil {
			t.Fatal("Expected error got nil %v", pe)
		}

		if pe.Id != e.Id {
			t.Fatal("Expected %s got %s", e.Id, pe.Id)
		}

		if pe.Detail != e.Detail {
			t.Fatal("Expected %s got %s", e.Detail, pe.Detail)
		}

		if pe.Code != e.Code {
			t.Fatal("Expected %s got %s", e.Code, pe.Code)
		}

		if pe.Status != e.Status {
			t.Fatal("Expected %s got %s", e.Status, pe.Status)
		}
	}
}
