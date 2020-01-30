package file

import (
	"testing"

	"github.com/micro/go-micro/v2/config/source"
)

func TestFormat(t *testing.T) {
	opts := source.NewOptions()
	e := opts.Encoder

	testCases := []struct {
		p string
		f string
	}{
		{"/foo/bar.json", "json"},
		{"/foo/bar.yaml", "yaml"},
		{"/foo/bar.xml", "xml"},
		{"/foo/bar.conf.ini", "ini"},
		{"conf", e.String()},
	}

	for _, d := range testCases {
		f := format(d.p, e)
		if f != d.f {
			t.Fatalf("%s: expected %s got %s", d.p, d.f, f)
		}
	}

}
