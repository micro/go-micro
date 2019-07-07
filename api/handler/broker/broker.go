// Package broker provides a go-micro/broker handler
package broker

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/micro/go-micro/api/handler"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/util/log"
)

const (
	Handler = "broker"

	pingTime      = (readDeadline * 9) / 10
	readLimit     = 16384
	readDeadline  = 60 * time.Second
	writeDeadline = 10 * time.Second
)

type brokerHandler struct {
	opts handler.Options
	u    websocket.Upgrader
}

type conn struct {
	b     broker.Broker
	cType string
	topic string
	queue string
	exit  chan bool

	sync.Mutex
	ws *websocket.Conn
}

var (
	once        sync.Once
	contentType = "text/plain"
)

func checkOrigin(r *http.Request) bool {
	origin := r.Header["Origin"]
	if len(origin) == 0 {
		return true
	}
	u, err := url.Parse(origin[0])
	if err != nil {
		return false
	}
	return u.Host == r.Host
}

func (c *conn) close() {
	select {
	case <-c.exit:
		return
	default:
		close(c.exit)
	}
}

func (c *conn) readLoop() {
	defer func() {
		c.close()
		c.ws.Close()
	}()

	// set read limit/deadline
	c.ws.SetReadLimit(readLimit)
	c.ws.SetReadDeadline(time.Now().Add(readDeadline))

	// set close handler
	ch := c.ws.CloseHandler()
	c.ws.SetCloseHandler(func(code int, text string) error {
		err := ch(code, text)
		c.close()
		return err
	})

	// set pong handler
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		c.b.Publish(c.topic, &broker.Message{
			Header: map[string]string{"Content-Type": c.cType},
			Body:   message,
		})
	}
}

func (c *conn) write(mType int, data []byte) error {
	c.Lock()
	c.ws.SetWriteDeadline(time.Now().Add(writeDeadline))
	err := c.ws.WriteMessage(mType, data)
	c.Unlock()
	return err
}

func (c *conn) writeLoop() {
	ticker := time.NewTicker(pingTime)

	var opts []broker.SubscribeOption

	if len(c.queue) > 0 {
		opts = append(opts, broker.Queue(c.queue))
	}

	subscriber, err := c.b.Subscribe(c.topic, func(p broker.Event) error {
		b, err := json.Marshal(p.Message())
		if err != nil {
			return nil
		}
		return c.write(websocket.TextMessage, b)
	}, opts...)

	defer func() {
		subscriber.Unsubscribe()
		ticker.Stop()
		c.ws.Close()
	}()

	if err != nil {
		log.Log(err.Error())
		return
	}

	for {
		select {
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		case <-c.exit:
			return
		}
	}
}

func (b *brokerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	br := b.opts.Service.Client().Options().Broker

	// Setup the broker
	once.Do(func() {
		br.Init()
		br.Connect()
	})

	// Parse
	r.ParseForm()
	topic := r.Form.Get("topic")

	// Can't do anything without a topic
	if len(topic) == 0 {
		http.Error(w, "Topic not specified", 400)
		return
	}

	// Post assumed to be Publish
	if r.Method == "POST" {
		// Create a broker message
		msg := &broker.Message{
			Header: make(map[string]string),
		}

		// Set header
		for k, v := range r.Header {
			msg.Header[k] = strings.Join(v, ", ")
		}

		// Read body
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Set body
		msg.Body = b

		// Publish
		br.Publish(topic, msg)
		return
	}

	// now back to our regularly scheduled programming

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	queue := r.Form.Get("queue")

	ws, err := b.u.Upgrade(w, r, nil)
	if err != nil {
		log.Log(err.Error())
		return
	}

	cType := r.Header.Get("Content-Type")
	if len(cType) == 0 {
		cType = contentType
	}

	c := &conn{
		b:     br,
		cType: cType,
		topic: topic,
		queue: queue,
		exit:  make(chan bool),
		ws:    ws,
	}

	go c.writeLoop()
	c.readLoop()
}

func (b *brokerHandler) String() string {
	return "broker"
}

func NewHandler(opts ...handler.Option) handler.Handler {
	return &brokerHandler{
		u: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		opts: handler.NewOptions(opts...),
	}
}

func WithCors(cors map[string]bool, opts ...handler.Option) handler.Handler {
	return &brokerHandler{
		u: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				if origin := r.Header.Get("Origin"); cors[origin] {
					return true
				} else if len(origin) > 0 && cors["*"] {
					return true
				} else if checkOrigin(r) {
					return true
				}
				return false
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		opts: handler.NewOptions(opts...),
	}
}
