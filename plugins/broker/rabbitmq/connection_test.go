package rabbitmq

import (
	"crypto/tls"
	"errors"
	"testing"

	"github.com/streadway/amqp"
)

func TestNewRabbitMQConnURL(t *testing.T) {
	testcases := []struct {
		title string
		urls  []string
		want  string
	}{
		{"Multiple URLs", []string{"amqp://example.com/one", "amqp://example.com/two"}, "amqp://example.com/one"},
		{"Insecure URL", []string{"amqp://example.com"}, "amqp://example.com"},
		{"Secure URL", []string{"amqps://example.com"}, "amqps://example.com"},
		{"Invalid URL", []string{"http://example.com"}, DefaultRabbitURL},
		{"No URLs", []string{}, DefaultRabbitURL},
	}

	for _, test := range testcases {
		conn := newRabbitMQConn(Exchange{Name: "exchange"}, test.urls, 0, false)

		if have, want := conn.url, test.want; have != want {
			t.Errorf("%s: invalid url, want %q, have %q", test.title, want, have)
		}
	}
}

func TestTryToConnectTLS(t *testing.T) {
	var (
		dialCount, dialTLSCount int

		err = errors.New("stop connect here")
	)

	dialConfig = func(_ string, c amqp.Config) (*amqp.Connection, error) {

		if c.TLSClientConfig != nil {
			dialTLSCount++
			return nil, err
		}

		dialCount++
		return nil, err
	}

	testcases := []struct {
		title      string
		url        string
		secure     bool
		amqpConfig *amqp.Config
		wantTLS    bool
	}{
		{"unsecure url, secure false, no tls config", "amqp://example.com", false, nil, false},
		{"secure url, secure false, no tls config", "amqps://example.com", false, nil, true},
		{"unsecure url, secure true, no tls config", "amqp://example.com", true, nil, true},
		{"unsecure url, secure false, tls config", "amqp://example.com", false, &amqp.Config{TLSClientConfig: &tls.Config{}}, true},
	}

	for _, test := range testcases {
		dialCount, dialTLSCount = 0, 0

		conn := newRabbitMQConn(Exchange{Name: "exchange"}, []string{test.url}, 0, false)
		conn.tryConnect(test.secure, test.amqpConfig)

		have := dialCount
		if test.wantTLS {
			have = dialTLSCount
		}

		if have != 1 {
			t.Errorf("%s: used wrong dialer, Dial called %d times, DialTLS called %d times", test.title, dialCount, dialTLSCount)
		}
	}
}

func TestNewRabbitMQPrefetch(t *testing.T) {
	testcases := []struct {
		title          string
		urls           []string
		prefetchCount  int
		prefetchGlobal bool
	}{
		{"Multiple URLs", []string{"amqp://example.com/one", "amqp://example.com/two"}, 1, true},
		{"Insecure URL", []string{"amqp://example.com"}, 1, true},
		{"Secure URL", []string{"amqps://example.com"}, 1, true},
		{"Invalid URL", []string{"http://example.com"}, 1, true},
		{"No URLs", []string{}, 1, true},
	}

	for _, test := range testcases {
		conn := newRabbitMQConn(Exchange{Name: "exchange"}, test.urls, test.prefetchCount, test.prefetchGlobal)

		if have, want := conn.prefetchCount, test.prefetchCount; have != want {
			t.Errorf("%s: invalid prefetch count, want %d, have %d", test.title, want, have)
		}

		if have, want := conn.prefetchGlobal, test.prefetchGlobal; have != want {
			t.Errorf("%s: invalid prefetch global setting, want %t, have %t", test.title, want, have)
		}
	}
}
