package tunnel

import (
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/log"
)

var (
	// DiscoverTime sets the time at which we fire discover messages
	DiscoverTime = 60 * time.Second
	// KeepAliveTime defines time interval we send keepalive messages to outbound links
	KeepAliveTime = 30 * time.Second
	// ReconnectTime defines time interval we periodically attempt to reconnect dead links
	ReconnectTime = 5 * time.Second
)

// tun represents a network tunnel
type tun struct {
	options Options

	sync.RWMutex

	// the unique id for this tunnel
	id string

	// tunnel token for authentication
	token string

	// to indicate if we're connected or not
	connected bool

	// the send channel for all messages
	send chan *message

	// close channel
	closed chan bool

	// a map of sessions based on Micro-Tunnel-Channel
	sessions map[string]*session

	// outbound links
	links map[string]*link

	// listener
	listener transport.Listener
}

// create new tunnel on top of a link
func newTunnel(opts ...Option) *tun {
	options := DefaultOptions()
	for _, o := range opts {
		o(&options)
	}

	return &tun{
		options:  options,
		id:       options.Id,
		token:    options.Token,
		send:     make(chan *message, 128),
		closed:   make(chan bool),
		sessions: make(map[string]*session),
		links:    make(map[string]*link),
	}
}

// Init initializes tunnel options
func (t *tun) Init(opts ...Option) error {
	t.Lock()
	defer t.Unlock()
	for _, o := range opts {
		o(&t.options)
	}
	return nil
}

// getSession returns a session from the internal session map.
// It does this based on the Micro-Tunnel-Channel and Micro-Tunnel-Session
func (t *tun) getSession(channel, session string) (*session, bool) {
	// get the session
	t.RLock()
	s, ok := t.sessions[channel+session]
	t.RUnlock()
	return s, ok
}

func (t *tun) delSession(channel, session string) {
	t.Lock()
	delete(t.sessions, channel+session)
	t.Unlock()
}

// listChannels returns a list of listening channels
func (t *tun) listChannels() []string {
	t.RLock()
	defer t.RUnlock()

	var channels []string
	for _, session := range t.sessions {
		if session.session != "listener" {
			continue
		}
		channels = append(channels, session.channel)
	}
	return channels
}

// newSession creates a new session and saves it
func (t *tun) newSession(channel, sessionId string) (*session, bool) {
	// new session
	s := &session{
		tunnel:  t.id,
		channel: channel,
		session: sessionId,
		closed:  make(chan bool),
		recv:    make(chan *message, 128),
		send:    t.send,
		wait:    make(chan bool),
		errChan: make(chan error, 1),
	}

	// save session
	t.Lock()
	_, ok := t.sessions[channel+sessionId]
	if ok {
		// session already exists
		t.Unlock()
		return nil, false
	}

	t.sessions[channel+sessionId] = s
	t.Unlock()

	// return session
	return s, true
}

// TODO: use tunnel id as part of the session
func (t *tun) newSessionId() string {
	return uuid.New().String()
}

func (t *tun) announce(channel, session string, link *link) {
	// create the "announce" response message for a discover request
	msg := &transport.Message{
		Header: map[string]string{
			"Micro-Tunnel":         "announce",
			"Micro-Tunnel-Id":      t.id,
			"Micro-Tunnel-Channel": channel,
			"Micro-Tunnel-Session": session,
			"Micro-Tunnel-Link":    link.id,
			"Micro-Tunnel-Token":   t.token,
		},
	}

	// if no channel is present we've been asked to discover all channels
	if len(channel) == 0 {
		// get the list of channels
		channels := t.listChannels()

		// if there are no channels continue
		if len(channels) == 0 {
			return
		}

		// create a list of channels as comma separated list
		channel = strings.Join(channels, ",")
		// set channels as header
		msg.Header["Micro-Tunnel-Channel"] = channel
	} else {
		// otherwise look for a single channel mapping
		// looking for existing mapping as a listener
		_, exists := t.getSession(channel, "listener")
		if !exists {
			return
		}
	}

	log.Debugf("Tunnel sending announce for discovery of channel(s) %s", channel)

	// send back the announcement
	if err := link.Send(msg); err != nil {
		log.Debugf("Tunnel failed to send announcement for channel(s) %s message: %v", channel, err)
	}
}

// monitor monitors outbound links and attempts to reconnect to the failed ones
func (t *tun) monitor() {
	reconnect := time.NewTicker(ReconnectTime)
	defer reconnect.Stop()

	for {
		select {
		case <-t.closed:
			return
		case <-reconnect.C:
			t.RLock()

			var delLinks []string
			// check the link status and purge dead links
			for node, link := range t.links {
				// check link status
				switch link.Status() {
				case "closed":
					delLinks = append(delLinks, node)
				case "error":
					delLinks = append(delLinks, node)
				}
			}

			t.RUnlock()

			// delete the dead links
			if len(delLinks) > 0 {
				t.Lock()
				for _, node := range delLinks {
					log.Debugf("Tunnel deleting dead link for %s", node)
					link := t.links[node]
					link.Close()
					delete(t.links, node)
				}
				t.Unlock()
			}

			// check current link status
			var connect []string

			// build list of unknown nodes to connect to
			t.RLock()
			for _, node := range t.options.Nodes {
				if _, ok := t.links[node]; !ok {
					connect = append(connect, node)
				}
			}
			t.RUnlock()

			for _, node := range connect {
				// create new link
				link, err := t.setupLink(node)
				if err != nil {
					log.Debugf("Tunnel failed to setup node link to %s: %v", node, err)
					continue
				}
				// save the link
				t.Lock()
				t.links[node] = link
				t.Unlock()
			}
		}
	}
}

// process outgoing messages sent by all local sessions
func (t *tun) process() {
	// manage the send buffer
	// all pseudo sessions throw everything down this
	for {
		select {
		case msg := <-t.send:
			newMsg := &transport.Message{
				Header: make(map[string]string),
			}

			// set the data
			if msg.data != nil {
				for k, v := range msg.data.Header {
					newMsg.Header[k] = v
				}
				newMsg.Body = msg.data.Body
			}

			// set message head
			newMsg.Header["Micro-Tunnel"] = msg.typ

			// set the tunnel id on the outgoing message
			newMsg.Header["Micro-Tunnel-Id"] = msg.tunnel

			// set the tunnel channel on the outgoing message
			newMsg.Header["Micro-Tunnel-Channel"] = msg.channel

			// set the session id
			newMsg.Header["Micro-Tunnel-Session"] = msg.session

			// set the tunnel token
			newMsg.Header["Micro-Tunnel-Token"] = t.token

			// send the message via the interface
			t.RLock()

			if len(t.links) == 0 {
				log.Debugf("No links to send message type: %s channel: %s", msg.typ, msg.channel)
			}

			var sent bool
			var err error
			var sendTo []*link

			// build the list of links ot send to
			for node, link := range t.links {
				// if the link is not connected skip it
				if !link.connected {
					log.Debugf("Link for node %s not connected", node)
					err = errors.New("link not connected")
					continue
				}

				// if the link was a loopback accepted connection
				// and the message is being sent outbound via
				// a dialled connection don't use this link
				if link.loopback && msg.outbound {
					err = errors.New("link is loopback")
					continue
				}

				// if the message was being returned by the loopback listener
				// send it back up the loopback link only
				if msg.loopback && !link.loopback {
					err = errors.New("link is not loopback")
					continue
				}

				// check the multicast mappings
				if msg.multicast {
					link.RLock()
					_, ok := link.channels[msg.channel]
					link.RUnlock()
					// channel mapping not found in link
					if !ok {
						continue
					}
				} else {
					// if we're picking the link check the id
					// this is where we explicitly set the link
					// in a message received via the listen method
					if len(msg.link) > 0 && link.id != msg.link {
						err = errors.New("link not found")
						continue
					}
				}

				// add to link list
				sendTo = append(sendTo, link)
			}

			t.RUnlock()

			// send the message
			for _, link := range sendTo {
				// send the message via the current link
				log.Debugf("Sending %+v to %s", newMsg.Header, link.Remote())

				if errr := link.Send(newMsg); errr != nil {
					log.Debugf("Tunnel error sending %+v to %s: %v", newMsg.Header, link.Remote(), errr)
					err = errors.New(errr.Error())
					t.delLink(link.Remote())
					continue
				}

				// is sent
				sent = true

				// keep sending broadcast messages
				if msg.broadcast || msg.multicast {
					continue
				}

				// break on unicast
				break
			}

			var gerr error

			// set the error if not sent
			if !sent {
				gerr = err
			}

			// skip if its not been set
			if msg.errChan == nil {
				continue
			}

			// return error non blocking
			select {
			case msg.errChan <- gerr:
			default:
			}
		case <-t.closed:
			return
		}
	}
}

func (t *tun) delLink(remote string) {
	t.Lock()
	defer t.Unlock()

	// get the link
	for id, link := range t.links {
		if link.id != remote {
			continue
		}
		// close and delete
		log.Debugf("Tunnel deleting link node: %s remote: %s", id, link.Remote())
		link.Close()
		delete(t.links, id)
	}
}

// process incoming messages
func (t *tun) listen(link *link) {
	// remove the link on exit
	defer func() {
		t.delLink(link.Remote())
	}()

	// let us know if its a loopback
	var loopback bool

	for {
		// process anything via the net interface
		msg := new(transport.Message)
		if err := link.Recv(msg); err != nil {
			log.Debugf("Tunnel link %s receive error: %#v", link.Remote(), err)
			return
		}

		// always ensure we have the correct auth token
		// TODO: segment the tunnel based on token
		// e.g use it as the basis
		token := msg.Header["Micro-Tunnel-Token"]
		if token != t.token {
			log.Debugf("Tunnel link %s received invalid token %s", token)
			return
		}

		// message type
		mtype := msg.Header["Micro-Tunnel"]
		// the tunnel id
		id := msg.Header["Micro-Tunnel-Id"]
		// the tunnel channel
		channel := msg.Header["Micro-Tunnel-Channel"]
		// the session id
		sessionId := msg.Header["Micro-Tunnel-Session"]

		// if its not connected throw away the link
		// the first message we process needs to be connect
		if !link.connected && mtype != "connect" {
			log.Debugf("Tunnel link %s not connected", link.id)
			return
		}

		switch mtype {
		case "connect":
			log.Debugf("Tunnel link %s received connect message", link.Remote())

			link.Lock()
			// are we connecting to ourselves?
			if id == t.id {
				link.loopback = true
				loopback = true
			}

			// set to remote node
			link.id = link.Remote()
			// set as connected
			link.connected = true
			link.Unlock()

			// save the link once connected
			t.Lock()
			t.links[link.Remote()] = link
			t.Unlock()

			// send back a discovery
			go t.announce("", "", link)
			// nothing more to do
			continue
		case "close":
			// TODO: handle the close message
			// maybe report io.EOF or kill the link

			// close the link entirely
			if len(channel) == 0 {
				log.Debugf("Tunnel link %s received close message", link.Remote())
				return
			}

			// the entire listener was closed so remove it from the mapping
			if sessionId == "listener" {
				link.Lock()
				delete(link.channels, channel)
				link.Unlock()
				continue
			}

			// try get the dialing socket
			s, exists := t.getSession(channel, sessionId)
			if exists {
				// close and continue
				s.Close()
				continue
			}
			// otherwise its a session mapping of sorts
		case "keepalive":
			log.Debugf("Tunnel link %s received keepalive", link.Remote())
			link.Lock()
			// save the keepalive
			link.lastKeepAlive = time.Now()
			link.Unlock()
			continue
		// a new connection dialled outbound
		case "open":
			log.Debugf("Tunnel link %s received open %s %s", link.id, channel, sessionId)
			// we just let it pass through to be processed
		// an accept returned by the listener
		case "accept":
			s, exists := t.getSession(channel, sessionId)
			// we don't need this
			if exists && s.multicast {
				s.accepted = true
				continue
			}
			if exists && s.accepted {
				continue
			}
		// a continued session
		case "session":
			// process message
			log.Debugf("Received %+v from %s", msg.Header, link.Remote())
		// an announcement of a channel listener
		case "announce":
			// process the announcement
			channels := strings.Split(channel, ",")

			// update mapping in the link
			link.Lock()
			for _, channel := range channels {
				link.channels[channel] = time.Now()
			}
			link.Unlock()

			// this was an announcement not intended for anything
			if sessionId == "listener" || sessionId == "" {
				continue
			}

			// get the session that asked for the discovery
			s, exists := t.getSession(channel, sessionId)
			if exists {
				// don't bother it's already discovered
				if s.discovered {
					continue
				}

				// send the announce back to the caller
				s.recv <- &message{
					typ:     "announce",
					tunnel:  id,
					channel: channel,
					session: sessionId,
					link:    link.id,
				}
			}
			continue
		case "discover":
			// send back an announcement
			go t.announce(channel, sessionId, link)
			continue
		default:
			// blackhole it
			continue
		}

		// strip tunnel message header
		for k, _ := range msg.Header {
			if strings.HasPrefix(k, "Micro-Tunnel") {
				delete(msg.Header, k)
			}
		}

		// if the session id is blank there's nothing we can do
		// TODO: check this is the case, is there any reason
		// why we'd have a blank session? Is the tunnel
		// used for some other purpose?
		if len(channel) == 0 || len(sessionId) == 0 {
			continue
		}

		var s *session
		var exists bool

		// If its a loopback connection then we've enabled link direction
		// listening side is used for listening, the dialling side for dialling
		switch {
		case loopback, mtype == "open":
			s, exists = t.getSession(channel, "listener")
		// only return accept to the session
		case mtype == "accept":
			log.Debugf("Received accept message for %s %s", channel, sessionId)
			s, exists = t.getSession(channel, sessionId)
			if exists && s.accepted {
				continue
			}
		default:
			// get the session based on the tunnel id and session
			// this could be something we dialed in which case
			// we have a session for it otherwise its a listener
			s, exists = t.getSession(channel, sessionId)
			if !exists {
				// try get it based on just the tunnel id
				// the assumption here is that a listener
				// has no session but its set a listener session
				s, exists = t.getSession(channel, "listener")
			}
		}

		// bail if no session or listener has been found
		if !exists {
			log.Debugf("Tunnel skipping no session %s %s exists", channel, sessionId)
			// drop it, we don't care about
			// messages we don't know about
			continue
		}

		// is the session closed?
		select {
		case <-s.closed:
			// closed
			delete(t.sessions, channel)
			continue
		default:
			// process
		}

		log.Debugf("Tunnel using channel %s session %s", s.channel, s.session)

		// is the session new?
		select {
		// if its new the session is actually blocked waiting
		// for a connection. so we check if its waiting.
		case <-s.wait:
		// if its waiting e.g its new then we close it
		default:
			// set remote address of the session
			s.remote = msg.Header["Remote"]
			close(s.wait)
		}

		// construct a new transport message
		tmsg := &transport.Message{
			Header: msg.Header,
			Body:   msg.Body,
		}

		// construct the internal message
		imsg := &message{
			tunnel:   id,
			typ:      mtype,
			channel:  channel,
			session:  sessionId,
			data:     tmsg,
			link:     link.id,
			loopback: loopback,
			errChan:  make(chan error, 1),
		}

		// append to recv backlog
		// we don't block if we can't pass it on
		select {
		case s.recv <- imsg:
		default:
		}
	}
}

// discover sends channel discover requests periodically
func (t *tun) discover(link *link) {
	tick := time.NewTicker(DiscoverTime)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			// send a discovery message to all links
			if err := link.Send(&transport.Message{
				Header: map[string]string{
					"Micro-Tunnel":       "discover",
					"Micro-Tunnel-Id":    t.id,
					"Micro-Tunnel-Token": t.token,
				},
			}); err != nil {
				log.Debugf("Tunnel failed to send discover to link %s: %v", link.id, err)
			}
		case <-link.closed:
			return
		case <-t.closed:
			return
		}
	}
}

// keepalive periodically sends keepalive messages to link
func (t *tun) keepalive(link *link) {
	keepalive := time.NewTicker(KeepAliveTime)
	defer keepalive.Stop()

	for {
		select {
		case <-t.closed:
			return
		case <-link.closed:
			return
		case <-keepalive.C:
			// send keepalive message
			log.Debugf("Tunnel sending keepalive to link: %v", link.Remote())
			if err := link.Send(&transport.Message{
				Header: map[string]string{
					"Micro-Tunnel":       "keepalive",
					"Micro-Tunnel-Id":    t.id,
					"Micro-Tunnel-Token": t.token,
				},
			}); err != nil {
				log.Debugf("Error sending keepalive to link %v: %v", link.Remote(), err)
				t.delLink(link.Remote())
				return
			}
		}
	}
}

// setupLink connects to node and returns link if successful
// It returns error if the link failed to be established
func (t *tun) setupLink(node string) (*link, error) {
	log.Debugf("Tunnel setting up link: %s", node)
	c, err := t.options.Transport.Dial(node)
	if err != nil {
		log.Debugf("Tunnel failed to connect to %s: %v", node, err)
		return nil, err
	}
	log.Debugf("Tunnel connected to %s", node)

	// send the first connect message
	if err := c.Send(&transport.Message{
		Header: map[string]string{
			"Micro-Tunnel":       "connect",
			"Micro-Tunnel-Id":    t.id,
			"Micro-Tunnel-Token": t.token,
		},
	}); err != nil {
		return nil, err
	}

	// create a new link
	link := newLink(c)
	// set link id to remote side
	link.id = c.Remote()
	// we made the outbound connection
	// and sent the connect message
	link.connected = true

	// process incoming messages
	go t.listen(link)

	// start keepalive monitor
	go t.keepalive(link)

	// discover things on the remote side
	go t.discover(link)

	return link, nil
}

// connect the tunnel to all the nodes and listen for incoming tunnel connections
func (t *tun) connect() error {
	l, err := t.options.Transport.Listen(t.options.Address)
	if err != nil {
		return err
	}

	// save the listener
	t.listener = l

	go func() {
		// accept inbound connections
		err := l.Accept(func(sock transport.Socket) {
			log.Debugf("Tunnel accepted connection from %s", sock.Remote())

			// create a new link
			link := newLink(sock)

			// start keepalive monitor
			go t.keepalive(link)

			// discover things on the remote side
			go t.discover(link)

			// listen for inbound messages.
			// only save the link once connected.
			// we do this inside liste
			t.listen(link)
		})

		t.RLock()
		defer t.RUnlock()

		// still connected but the tunnel died
		if err != nil && t.connected {
			log.Logf("Tunnel listener died: %v", err)
		}
	}()

	for _, node := range t.options.Nodes {
		// skip zero length nodes
		if len(node) == 0 {
			continue
		}

		// connect to node and return link
		link, err := t.setupLink(node)
		if err != nil {
			log.Debugf("Tunnel failed to establish node link to %s: %v", node, err)
			continue
		}

		// save the link
		t.links[node] = link
	}

	// process outbound messages to be sent
	// process sends to all links
	go t.process()

	// monitor links
	go t.monitor()

	return nil
}

// Connect the tunnel
func (t *tun) Connect() error {
	t.Lock()
	defer t.Unlock()

	// already connected
	if t.connected {
		return nil
	}

	// send the connect message
	if err := t.connect(); err != nil {
		return err
	}

	// set as connected
	t.connected = true
	// create new close channel
	t.closed = make(chan bool)

	return nil
}

func (t *tun) close() error {
	// close all the sessions
	for id, s := range t.sessions {
		s.Close()
		delete(t.sessions, id)
	}

	// close all the links
	for node, link := range t.links {
		link.Send(&transport.Message{
			Header: map[string]string{
				"Micro-Tunnel":       "close",
				"Micro-Tunnel-Id":    t.id,
				"Micro-Tunnel-Token": t.token,
			},
		})
		link.Close()
		delete(t.links, node)
	}

	// close the listener
	// this appears to be blocking
	return t.listener.Close()
}

func (t *tun) Address() string {
	t.RLock()
	defer t.RUnlock()

	if !t.connected {
		return t.options.Address
	}

	return t.listener.Addr()
}

// Close the tunnel
func (t *tun) Close() error {
	t.Lock()
	defer t.Unlock()

	if !t.connected {
		return nil
	}

	log.Debug("Tunnel closing")

	select {
	case <-t.closed:
		return nil
	default:
		close(t.closed)
		t.connected = false

		// send a close message
		// we don't close the link
		// just the tunnel
		return t.close()
	}

	return nil
}

// Dial an address
func (t *tun) Dial(channel string, opts ...DialOption) (Session, error) {
	log.Debugf("Tunnel dialing %s", channel)
	c, ok := t.newSession(channel, t.newSessionId())
	if !ok {
		return nil, errors.New("error dialing " + channel)
	}
	// set remote
	c.remote = channel
	// set local
	c.local = "local"
	// outbound session
	c.outbound = true

	// get opts
	options := DialOptions{
		Timeout: DefaultDialTimeout,
	}

	for _, o := range opts {
		o(&options)
	}

	// set the multicast option
	c.multicast = options.Multicast
	// set the dial timeout
	c.timeout = options.Timeout

	now := time.Now()

	after := func() time.Duration {
		d := time.Since(now)
		// dial timeout minus time since
		wait := options.Timeout - d
		if wait < time.Duration(0) {
			return time.Duration(0)
		}
		return wait
	}

	var links []string

	// non multicast so we need to find the link
	t.RLock()
	for _, link := range t.links {
		// use the link specified it its available
		if id := options.Link; len(id) > 0 && link.id != id {
			continue
		}

		link.RLock()
		_, ok := link.channels[channel]
		link.RUnlock()

		// we have at least one channel mapping
		if ok {
			c.discovered = true
			links = append(links, link.id)
		}
	}
	t.RUnlock()

	// link not found
	if len(links) == 0 && len(options.Link) > 0 {
		return nil, ErrLinkNotFound
	}

	// discovered so set the link if not multicast
	// TODO: pick the link efficiently based
	// on link status and saturation.
	if c.discovered && !c.multicast {
		// set the link
		i := rand.Intn(len(links))
		c.link = links[i]
	}

	// shit fuck
	if !c.discovered {
		// create a new discovery message for this channel
		msg := c.newMessage("discover")
		msg.broadcast = true
		msg.outbound = true
		msg.link = ""

		// send the discovery message
		t.send <- msg

		select {
		case <-time.After(after()):
			return nil, ErrDialTimeout
		case err := <-c.errChan:
			if err != nil {
				return nil, err
			}
		}

		var err error

		// set a dialTimeout
		dialTimeout := after()

		// set a shorter delay for multicast
		if c.multicast {
			// shorten this
			dialTimeout = time.Millisecond * 500
		}

		// wait for announce
		select {
		case msg := <-c.recv:
			if msg.typ != "announce" {
				err = ErrDiscoverChan
			}
		case <-time.After(dialTimeout):
			err = ErrDialTimeout
		}

		// if its multicast just go ahead because this is best effort
		if c.multicast {
			c.discovered = true
			c.accepted = true
			return c, nil
		}

		// otherwise return an error
		if err != nil {
			return nil, err
		}

		// set discovered to true
		c.discovered = true
	}

	// a unicast session so we call "open" and wait for an "accept"

	// try to open the session
	err := c.Open()
	if err != nil {
		// delete the session
		t.delSession(c.channel, c.session)
		return nil, err
	}

	return c, nil
}

// Accept a connection on the address
func (t *tun) Listen(channel string) (Listener, error) {
	log.Debugf("Tunnel listening on %s", channel)

	// create a new session by hashing the address
	c, ok := t.newSession(channel, "listener")
	if !ok {
		return nil, errors.New("already listening on " + channel)
	}

	delFunc := func() {
		t.delSession(channel, "listener")
	}

	// set remote. it will be replaced by the first message received
	c.remote = "remote"
	// set local
	c.local = channel

	tl := &tunListener{
		channel: channel,
		// the accept channel
		accept: make(chan *session, 128),
		// the channel to close
		closed: make(chan bool),
		// tunnel closed channel
		tunClosed: t.closed,
		// the listener session
		session: c,
		// delete session
		delFunc: delFunc,
	}

	// this kicks off the internal message processor
	// for the listener so it can create pseudo sessions
	// per session if they do not exist or pass messages
	// to the existign sessions
	go tl.process()

	// announces the listener channel to others
	go tl.announce()

	// return the listener
	return tl, nil
}

func (t *tun) Links() []Link {
	t.RLock()
	defer t.RUnlock()

	var links []Link

	for _, link := range t.links {
		links = append(links, link)
	}

	return links
}

func (t *tun) String() string {
	return "mucp"
}
