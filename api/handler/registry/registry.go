// Package registry is a go-micro/registry handler
package registry

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/micro/go-micro/api/handler"
	"github.com/micro/go-micro/registry"
)

const (
	Handler = "registry"

	pingTime      = (readDeadline * 9) / 10
	readLimit     = 16384
	readDeadline  = 60 * time.Second
	writeDeadline = 10 * time.Second
)

type registryHandler struct {
	opts handler.Options
	reg  registry.Registry
}

func (rh *registryHandler) add(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer r.Body.Close()

	var opts []registry.RegisterOption

	// parse ttl
	if ttl := r.Form.Get("ttl"); len(ttl) > 0 {
		d, err := time.ParseDuration(ttl)
		if err == nil {
			opts = append(opts, registry.RegisterTTL(d))
		}
	}

	var service *registry.Service
	err = json.Unmarshal(b, &service)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	err = rh.reg.Register(service, opts...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (rh *registryHandler) del(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer r.Body.Close()

	var service *registry.Service
	err = json.Unmarshal(b, &service)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	err = rh.reg.Deregister(service)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (rh *registryHandler) get(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	service := r.Form.Get("service")

	var s []*registry.Service
	var err error

	if len(service) == 0 {
		//
		upgrade := r.Header.Get("Upgrade")
		connect := r.Header.Get("Connection")

		// watch if websockets
		if upgrade == "websocket" && connect == "Upgrade" {
			rw, err := rh.reg.Watch()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			watch(rw, w, r)
			return
		}

		// otherwise list services
		s, err = rh.reg.ListServices()
	} else {
		s, err = rh.reg.GetService(service)
	}

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if s == nil || (len(service) > 0 && (len(s) == 0 || len(s[0].Name) == 0)) {
		http.Error(w, "Service not found", 404)
		return
	}

	b, err := json.Marshal(s)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Write(b)
}

func ping(ws *websocket.Conn, exit chan bool) {
	ticker := time.NewTicker(pingTime)

	for {
		select {
		case <-ticker.C:
			ws.SetWriteDeadline(time.Now().Add(writeDeadline))
			err := ws.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				return
			}
		case <-exit:
			return
		}
	}
}

func watch(rw registry.Watcher, w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// we need an exit chan
	exit := make(chan bool)

	defer func() {
		close(exit)
	}()

	// ping the socket
	go ping(ws, exit)

	for {
		// get next result
		r, err := rw.Next()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// write to client
		ws.SetWriteDeadline(time.Now().Add(writeDeadline))
		if err := ws.WriteJSON(r); err != nil {
			return
		}
	}
}

func (rh *registryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		rh.get(w, r)
	case "POST":
		rh.add(w, r)
	case "DELETE":
		rh.del(w, r)
	}
}

func (rh *registryHandler) String() string {
	return "registry"
}

func NewHandler(opts ...handler.Option) handler.Handler {
	options := handler.NewOptions(opts...)

	return &registryHandler{
		opts: options,
		reg:  options.Service.Client().Options().Registry,
	}
}
