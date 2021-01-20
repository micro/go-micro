package proxy

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/asim/go-micro/v3/broker"
	log "github.com/asim/go-micro/v3/logger"
)

const (
	pingTime      = (readDeadline * 9) / 10
	readLimit     = 16384
	readDeadline  = 60 * time.Second
	writeDeadline = 10 * time.Second
)

type subscriber struct {
	opts    broker.SubscribeOptions
	conn    *websocket.Conn
	handler broker.Handler
	topic   string
	exit    chan bool
}

func newSubscriber(url, topic string, h broker.Handler, opts broker.SubscribeOptions) (broker.Subscriber, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, make(http.Header))
	if err != nil {
		return nil, err
	}

	s := &subscriber{
		opts:    opts,
		conn:    conn,
		handler: h,
		topic:   topic,
		exit:    make(chan bool),
	}

	go s.run()
	go s.ping()

	return s, nil
}

func (s *subscriber) ping() {
	ticker := time.NewTicker(pingTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			err := s.conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				log.Errorf("subscriber error writing ping message: %v", err)
				return
			}
		case <-s.exit:
			return
		}
	}
}

func (s *subscriber) run() {
	// set read limit/deadline
	s.conn.SetReadLimit(readLimit)
	s.conn.SetReadDeadline(time.Now().Add(readDeadline))

	// set close handler
	ch := s.conn.CloseHandler()
	s.conn.SetCloseHandler(func(code int, text string) error {
		err := ch(code, text)
		s.Unsubscribe()
		return err
	})

	// set pong handler
	s.conn.SetPongHandler(func(string) error {
		s.conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	// read and execution loop
	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg *broker.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			// do what?
			log.Errorf("subscriber error unmarshaling message: %v", err)
			continue
		}
		p := &publication{topic: s.topic, message: msg}
		p.err = s.handler(p)
		if p.err != nil {
			log.Errorf("handler execution error: %v", err)
		}
	}
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Unsubscribe() error {
	select {
	case <-s.exit:
		return nil
	default:
		close(s.exit)
		return s.conn.Close()
	}
}
