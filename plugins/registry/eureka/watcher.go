package eureka

import (
	"errors"
	"time"

	"github.com/hudl/fargo"
	"github.com/micro/go-micro/v2/registry"
)

type eurekaWatcher struct {
	conn    fargoConnection
	exit    chan bool
	results chan *registry.Result
}

func newWatcher(conn fargoConnection, opts ...registry.WatchOption) registry.Watcher {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	w := &eurekaWatcher{
		conn:    conn,
		exit:    make(chan bool),
		results: make(chan *registry.Result),
	}

	// watch a single service
	if len(wo.Service) > 0 {
		done := make(chan struct{})
		ch := conn.ScheduleAppUpdates(wo.Service, false, done)
		go w.watch(ch, done)
		go func() {
			<-w.exit
			close(done)
		}()
		return w
	}

	// watch all services
	go w.poll()
	return w
}

func (e *eurekaWatcher) poll() {
	// list service ticker
	t := time.NewTicker(time.Second * 10)

	done := make(chan struct{})
	services := make(map[string]<-chan fargo.AppUpdate)

	for {
		select {
		case <-e.exit:
			close(done)
			return
		case <-t.C:
			apps, err := e.conn.GetApps()
			if err != nil {
				continue
			}
			for _, app := range apps {
				if _, ok := services[app.Name]; ok {
					continue
				}
				ch := e.conn.ScheduleAppUpdates(app.Name, false, done)
				services[app.Name] = ch
				go e.watch(ch, done)
			}
		}
	}
}

func (e *eurekaWatcher) watch(ch <-chan fargo.AppUpdate, done chan struct{}) {
	for {
		select {
		// exit on exit
		case <-e.exit:
			return
		// exit on done
		case <-done:
			return
		// process updates
		case u := <-ch:
			if u.Err != nil {
				continue
			}

			// process instances independently
			for _, instance := range u.App.Instances {
				var action string

				switch instance.Status {
				// update
				case fargo.UP:
					action = "update"
				// delete
				case fargo.OUTOFSERVICE, fargo.UNKNOWN, fargo.DOWN:
					action = "delete"
				// skip
				default:
					continue
				}

				// construct the service with a single node
				service := appToService(&fargo.Application{
					Name:      u.App.Name,
					Instances: []*fargo.Instance{instance},
				})

				if len(service) == 0 {
					continue
				}

				// in case we get bounced during processing
				// check exit channels
				select {
				// send the update
				case e.results <- &registry.Result{Action: action, Service: service[0]}:
				case <-done:
					return
				case <-e.exit:
					return
				}
			}
		}
	}
}

func (e *eurekaWatcher) Next() (*registry.Result, error) {
	select {
	case <-e.exit:
		return nil, errors.New("watcher stopped")
	case r := <-e.results:
		return r, nil
	}
}

func (e *eurekaWatcher) Stop() {
	close(e.exit)
}
