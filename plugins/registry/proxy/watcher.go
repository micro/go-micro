package proxy

import (
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/registry"
)

const (
	pingTime      = (readDeadline * 9) / 10
	readLimit     = 16384
	readDeadline  = 60 * time.Second
	writeDeadline = 10 * time.Second
)

type watcher struct {
	exit chan bool
	conn *websocket.Conn
	res  chan *registry.Result
}

func (w *watcher) ping() {
	ticker := time.NewTicker(pingTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			err := w.conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				log.Errorf("watcher error writing ping message: %v", err)
				return
			}
		case <-w.exit:
			return
		}
	}
}

func (w *watcher) run() {
	// set read limit/deadline
	w.conn.SetReadLimit(readLimit)
	w.conn.SetReadDeadline(time.Now().Add(readDeadline))

	// set close handler
	ch := w.conn.CloseHandler()
	w.conn.SetCloseHandler(func(code int, text string) error {
		err := ch(code, text)
		w.Stop()
		return err
	})

	// set pong handler
	w.conn.SetPongHandler(func(string) error {
		w.conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	// read results and send to res channel
	for {
		var res *registry.Result
		if err := w.conn.ReadJSON(&res); err != nil {
			log.Errorf("error unmarshaling result: %v", err)
			return
		}
		select {
		case w.res <- res:
		case <-w.exit:
			return
		}
	}
}

func (w *watcher) Next() (*registry.Result, error) {
	select {
	case <-w.exit:
		return nil, errors.New("result chan stopped")
	case r, ok := <-w.res:
		if !ok {
			return nil, errors.New("result chan stopped")
		}
		return r, nil
	}
}

func (w *watcher) Stop() {
	select {
	case <-w.exit:
		return
	default:
		close(w.exit)
	}
}

func newWatcher(url string) (registry.Watcher, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, make(http.Header))
	if err != nil {
		return nil, err
	}
	w := &watcher{
		conn: conn,
		exit: make(chan bool),
		res:  make(chan *registry.Result),
	}
	go w.ping()
	go w.run()
	return w, nil
}
