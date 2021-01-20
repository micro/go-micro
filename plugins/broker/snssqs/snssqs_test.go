package snssqs

import (
	"testing"

	"github.com/asim/go-micro/v3/broker"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		msg     *broker.Message
		wantErr bool
	}{
		{"Empty", &broker.Message{Body: []byte("")}, false},
		{"Accept1", &broker.Message{Body: []byte("\u0009")}, false},
		{"Accept2", &broker.Message{Body: []byte("\u000a")}, false},
		{"Accept3", &broker.Message{Body: []byte("\u000a\u0009")}, false},
		{"Reject1", &broker.Message{Body: []byte("\u0007")}, true},
		{"Reject2", &broker.Message{Body: []byte("\u0009\u0007")}, true},
		{"Reject3", &broker.Message{Body: []byte("\u0007\u0009")}, true},
		{"Longer1", &broker.Message{Body: []byte(
			"This is a longer valid unicode string that should test that workers process runes correctly.")}, false},
		{"Longer2", &broker.Message{Body: []byte(
			"This is a longer unicode string containing an invalid character \u0007 that should test the inverse.")}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateBody(tt.msg); (err != nil) != tt.wantErr {
				t.Errorf("ValidateBody() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
