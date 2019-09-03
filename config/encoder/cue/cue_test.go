package cue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_cueEncoder_Decode(t *testing.T) {

	type Cfg struct {
		Msg   string
		Place string
	}
	var vv Cfg

	type args struct {
		d []byte
		v interface{}
	}
	tests := []struct {
		name    string
		c       cueEncoder
		args    args
		wantErr bool
	}{
		{
			name: "test place holder",
			c:    cueEncoder{},
			args: args{
				d: []byte(`
msg:   "Hello \(place)!"
place: string | *"world" // "world" is the default.
`),
				v: &vv,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.Decode(tt.args.d, tt.args.v)
			assert.NoError(t, err)
			assert := assert.New(t)
			assert.Equal(vv.Msg, "Hello world!")
		})
	}
}
