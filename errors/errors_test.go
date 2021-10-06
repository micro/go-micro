package errors

import (
	er "errors"
	"net/http"
	"testing"
)

func TestFromError(t *testing.T) {
	err := NotFound("go.micro.test", "%s", "example")
	merr := FromError(err)
	if merr.Id != "go.micro.test" || merr.Code != 404 {
		t.Fatalf("invalid conversation %v != %v", err, merr)
	}
	err = er.New(err.Error())
	merr = FromError(err)
	if merr.Id != "go.micro.test" || merr.Code != 404 {
		t.Fatalf("invalid conversation %v != %v", err, merr)
	}
	merr = FromError(nil)
	if merr != nil {
		t.Fatalf("%v should be nil", merr)
	}
}

func TestEqual(t *testing.T) {
	err1 := NotFound("myid1", "msg1")
	err2 := NotFound("myid2", "msg2")

	if !Equal(err1, err2) {
		t.Fatal("errors must be equal")
	}

	err3 := er.New("my test err")
	if Equal(err1, err3) {
		t.Fatal("errors must be not equal")
	}

}

func TestErrors(t *testing.T) {
	testData := []*Error{
		{
			Id:     "test",
			Code:   500,
			Detail: "Internal server error",
			Status: http.StatusText(500),
		},
	}

	for _, e := range testData {
		ne := New(e.Id, e.Detail, e.Code)

		if e.Error() != ne.Error() {
			t.Fatalf("Expected %s got %s", e.Error(), ne.Error())
		}

		pe := Parse(ne.Error())

		if pe == nil {
			t.Fatalf("Expected error got nil %v", pe)
		}

		if pe.Id != e.Id {
			t.Fatalf("Expected %s got %s", e.Id, pe.Id)
		}

		if pe.Detail != e.Detail {
			t.Fatalf("Expected %s got %s", e.Detail, pe.Detail)
		}

		if pe.Code != e.Code {
			t.Fatalf("Expected %d got %d", e.Code, pe.Code)
		}

		if pe.Status != e.Status {
			t.Fatalf("Expected %s got %s", e.Status, pe.Status)
		}
	}
}

func TestAs(t *testing.T) {
	err := NotFound("go.micro.test", "%s", "example")
	merr, match := As(err)
	if !match {
		t.Fatalf("%v should convert to *Error", err)
	}
	if merr.Id != "go.micro.test" || merr.Code != 404 || merr.Detail != "example" {
		t.Fatalf("invalid conversation %v != %v", err, merr)
	}
	err = er.New(err.Error())
	merr, match = As(err)
	if match || merr != nil {
		t.Fatalf("%v should not convert to *Error", err)
	}
	merr, match = As(nil)
	if match || merr != nil {
		t.Fatalf("nil should not convert to *Error")
	}
}

func TestAppend(t *testing.T) {
	mError := NewMultiError()
	testData := []*Error{
		{
			Id:     "test1",
			Code:   500,
			Detail: "Internal server error",
			Status: http.StatusText(500),
		},
		{
			Id: 	"test2",
			Code:	400,
			Detail:	"Bad Request",
			Status: http.StatusText(400),
		},
		{
			Id:     "test3",
			Code:   404,
			Detail: "Not Found",
			Status: http.StatusText(404),	
		},
	}

	for _, e := range testData {
		mError.Append(&Error{
			Id: e.Id,
			Code: e.Code,
			Detail: e.Detail,
			Status: e.Status,
		})
	}

	if len(mError.Errors) != 3 {
		t.Fatalf("Expected 3 got %v", len(mError.Errors))
	}
}

func TestHasErrors(t *testing.T) {
	mError := NewMultiError()
	testData := []*Error{
		{
			Id:     "test1",
			Code:   500,
			Detail: "Internal server error",
			Status: http.StatusText(500),
		},
		{
			Id: 	"test2",
			Code:	400,
			Detail:	"Bad Request",
			Status: http.StatusText(400),
		},
		{
			Id:     "test3",
			Code:   404,
			Detail: "Not Found",
			Status: http.StatusText(404),	
		},
	}

	if mError.HasErrors() {
		t.Fatal("Expected no error")
	}

	for _, e := range testData {
		mError.Errors = append(mError.Errors, &Error{
			Id: e.Id,
			Code: e.Code,
			Detail: e.Detail,
			Status: e.Status,
		})
	}

	if !mError.HasErrors() {
		t.Fatal("Expected errors")
	}
}