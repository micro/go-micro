The Go Playground   Imports 
1
2
3
4
5
6
7
8
9
10
11
12
13
14
15
16
17
18
19
20
21
22
23
24
25
26
27
28
29
30
31
32
33
34
35
36
37
38
39
40
41
42
43
44
45
46
47
48
49
50
51
52
53
54
55
56
57
58
59
60
61
62
63
64
65
66
67
68
69
70
71
72
73
74
75
76
77
78
79
80
81
82
83
84
85
86
87
88
89
90
91
92
93
94
95
96
97
98
99
100
101
102
103
104
105
106
107
108
109
110
111
112
113
114
115
116
117
118
119
120
121
122
123
124
125
126
127
128
129
130
131
132
133
134
135
136
137
138
139
140
141
142
143
144
145
146
147
148
149
150
151
152
153
154
155
156
157
158
159
160
161
162
163
164
165
166
167
168
169
170
171
172
173
174
175
176
177
178
179
180
181
182
183
184
185
186
187
188
189
190
191
192
193
194
195
196
197
198
199
200
201
202
203
204
205
206
207
208
209
210
211
212
213
214
215
216
217
218
219
220
221
222
223
224
225
226
227
228
229
230
231
232
233
234
235
236
237
238
239
240
241
242
243
244
245
246
247
248
249
250
251
252
253
254
255
256
257
258
259
260
261
262
263
264
265
266
267
268
269
270
271
272
273
274
275
276
277
278
279
280
281
282
283
284
285
286
287
288
289
290
291
292
293
294
295
296
297
298
299
300
301
302
303
304
305
306
307
308
309
310
311
312
313
314
315
316
317
318
319
320
321
322
323
324
325
326
327
328
329
330
331
332
333
334
335
336
337
338
339
340
341
342
343
344
345
346
347
348
349
350
351
352
353
354
355
356
357
358
359
360
361
362
363
364
365
366
367
368
369
370
371
372
373
374
375
376
377
378
379
380
381
382
383
384
385
386
387
388
389
390
391
392
393
394
395
396
397
398
399
400
401
402
403
404
405
406
407
408
409
410
411
412
413
414
415
416
417
418
419
420
421
422
423
424
425
426
427
428
429
430
431
432
433
434
435
436
437
438
439
440
441
442
443
444
445
446
447
448
449
450
451
452
453
454
455
456
457
458
459
460
461
462
463
464
465
466
467
468
469
470
471
472
473
474
475
476
477
478
479
480
481
482
483
484
485
486
487
488
489
490
491
492
493
494
495
496
497
498
499
500
501
502
503
504
505
506
507
508
509
510
511
512
513
514
515
516
517
518
519
520
521
522
523
524
525
526
527
528
529
530
531
532
533
534
535
536
537
538
539
540
541
542
543
544
545
546
547
548
549
550
551
552
553
554
555
556
557
558
559
560
561
562
563
564
565
566
567
568
569
570
571
572
573
574
575
576
577
578
579
580
581
582
583
584
585
586
587
588
589
590
591
592
593
594
595
596
597
598
599
600
601
602
603
604
605
606
607
608
609
610
611
612
613
614
615
616
617
618
619
620
621
622
623
624
625
626
627
628
629
630
631
632
633
634
635
636
637
638
639
640
641
642
643
644
645
646
647
648
649
650
651
652
653
654
655
656
657
658
659
660
661
662
663
664
665
666
667
668
669
670
671
672
673
674
675
676
677
678
679
680
681
682
683
684
685
686
687
688
689
690
691
692
693
694
695
696
697
698
699
700
701
702
703
704
705
706
707
708
709
710
711
712
713
714
715
716
717
718
719
720
721
722
723
724
725
726
727
728
729
730
731
732
733
734
735
736
737
738
739
740
741
742
743
744
745
746
747
748
749
750
751
752
753
754
755
756
757
758
759
760
761
762
763
764
765
766
767
768
769
770
771
772
773
774
775
776
777
778
779
780
781
782
783
784
785
786
787
788
789
790
791
792
793
794
795
796
797
package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/codec"
	"github.com/micro/go-micro/metadata"
	"github.com/micro/go-micro/registry"
	"github.com/micro/go-micro/transport"
	"github.com/micro/go-micro/util/addr"
	log "github.com/micro/go-micro/util/log"
	mnet "github.com/micro/go-micro/util/net"
	"github.com/micro/go-micro/util/socket"
)

type rpcServer struct {
	router *router
	exit   chan chan error

	sync.RWMutex
	opts        Options
	handlers    map[string]Handler
	subscribers map[*subscriber][]broker.Subscriber
	// marks the serve as started
	started bool
	// used for first registration
	registered bool
	// graceful exit
	wg *sync.WaitGroup
}

func newRpcServer(opts ...Option) Server {
	options := newOptions(opts...)
	router := newRpcRouter()
	router.hdlrWrappers = options.HdlrWrappers

	return &rpcServer{
		opts:        options,
		router:      router,
		handlers:    make(map[string]Handler),
		subscribers: make(map[*subscriber][]broker.Subscriber),
		exit:        make(chan chan error),
		wg:          wait(options.Context),
	}
}

type rpcRouter struct {
	h func(context.Context, Request, interface{}) error
}

func (r rpcRouter) ServeRequest(ctx context.Context, req Request, rsp Response) error {
	return r.h(ctx, req, rsp)
}

// ServeConn serves a single connection
func (s *rpcServer) ServeConn(sock transport.Socket) {
	var wg sync.WaitGroup
	var mtx sync.RWMutex
	// streams are multiplexed on Micro-Stream or Micro-Id header
	sockets := make(map[string]*socket.Socket)

	defer func() {
		// wait till done
		wg.Wait()

		// close underlying socket
		sock.Close()

		// close the sockets
		mtx.Lock()
		for id, psock := range sockets {
			psock.Close()
			delete(sockets, id)
		}
		mtx.Unlock()

		// recover any panics
		if r := recover(); r != nil {
			log.Log("panic recovered: ", r)
			log.Log(string(debug.Stack()))
		}
	}()

	for {
		var msg transport.Message
		if err := sock.Recv(&msg); err != nil {
			return
		}

		// use Micro-Stream as the stream identifier
		// in the event its blank we'll always process
		// on the same socket
		id := msg.Header["Micro-Stream"]

		// if there's no stream id then its a standard request
		// use the Micro-Id
		if len(id) == 0 {
			id = msg.Header["Micro-Id"]
		}

		// we're starting processing
		wg.Add(1)

		// add to wait group if "wait" is opt-in
		if s.wg != nil {
			s.wg.Add(1)
		}

		// check we have an existing socket
		mtx.RLock()
		psock, ok := sockets[id]
		mtx.RUnlock()

		// got the socket
		if ok {
			// accept the message
			if err := psock.Accept(&msg); err != nil {
				// delete the socket
				mtx.Lock()
				delete(sockets, id)
				mtx.Unlock()
			}

			// done(1)
			if s.wg != nil {
				s.wg.Done()
			}

			wg.Done()

			// continue to the next message
			continue
		}

		// no socket was found
		psock = socket.New()
		psock.SetLocal(sock.Local())
		psock.SetRemote(sock.Remote())

		// load the socket
		psock.Accept(&msg)

		// save a new socket
		mtx.Lock()
		sockets[id] = psock
		mtx.Unlock()

		// now walk the usual path

		// we use this Timeout header to set a server deadline
		to := msg.Header["Timeout"]
		// we use this Content-Type header to identify the codec needed
		ct := msg.Header["Content-Type"]

		// copy the message headers
		hdr := make(map[string]string)
		for k, v := range msg.Header {
			hdr[k] = v
		}

		// set local/remote ips
		hdr["Local"] = sock.Local()
		hdr["Remote"] = sock.Remote()

		// create new context with the metadata
		ctx := metadata.NewContext(context.Background(), hdr)

		// set the timeout from the header if we have it
		if len(to) > 0 {
			if n, err := strconv.ParseUint(to, 10, 64); err == nil {
				ctx, _ = context.WithTimeout(ctx, time.Duration(n))
			}
		}

		// if there's no content type default it
		if len(ct) == 0 {
			msg.Header["Content-Type"] = DefaultContentType
			ct = DefaultContentType
		}

		// setup old protocol
		cf := setupProtocol(&msg)

		// no old codec
		if cf == nil {
			// TODO: needs better error handling
			var err error
			if cf, err = s.newCodec(ct); err != nil {
				sock.Send(&transport.Message{
					Header: map[string]string{
						"Content-Type": "text/plain",
					},
					Body: []byte(err.Error()),
				})

				if s.wg != nil {
					s.wg.Done()
				}

				wg.Done()

				return
			}
		}

		rcodec := newRpcCodec(&msg, psock, cf)
		protocol := rcodec.String()

		// check stream id
		var stream bool

		if v := getHeader("Micro-Stream", msg.Header); len(v) > 0 {
			stream = true
		}

		// internal request
		request := &rpcRequest{
			service:     getHeader("Micro-Service", msg.Header),
			method:      getHeader("Micro-Method", msg.Header),
			endpoint:    getHeader("Micro-Endpoint", msg.Header),
			contentType: ct,
			codec:       rcodec,
			header:      msg.Header,
			body:        msg.Body,
			socket:      psock,
			stream:      stream,
		}

		// internal response
		response := &rpcResponse{
			header: make(map[string]string),
			socket: psock,
			codec:  rcodec,
		}

		// set router
		r := Router(s.router)

		// if not nil use the router specified
		if s.opts.Router != nil {
			// create a wrapped function
			handler := func(ctx context.Context, req Request, rsp interface{}) error {
				return s.opts.Router.ServeRequest(ctx, req, rsp.(Response))
			}

			// execute the wrapper for it
			for i := len(s.opts.HdlrWrappers); i > 0; i-- {
				handler = s.opts.HdlrWrappers[i-1](handler)
			}

			// set the router
			r = rpcRouter{handler}
		}

		// wait for processing to exit
		wg.Add(1)

		// process the outbound messages from the socket
		go func(id string, psock *socket.Socket) {
			defer func() {
				// TODO: don't hack this but if its grpc just break out of the stream
				// We do this because the underlying connection is h2 and its a stream
				switch protocol {
				case "grpc":
					sock.Close()
				}

				wg.Done()
			}()

			for {
				// get the message from our internal handler/stream
				m := new(transport.Message)
				if err := psock.Process(m); err != nil {
					// delete the socket
					mtx.Lock()
					delete(sockets, id)
					mtx.Unlock()
					return
				}

				// send the message back over the socket
				if err := sock.Send(m); err != nil {
					return
				}
			}
		}(id, psock)

		// serve the request in a go routine as this may be a stream
		go func(id string, psock *socket.Socket) {
			defer psock.Close()

			// serve the actual request using the request router
			if serveRequestError := r.ServeRequest(ctx, request, response); serveRequestError != nil {
				// write an error response
				writeError := rcodec.Write(&codec.Message{
					Header: msg.Header,
					Error:  serveRequestError.Error(),
					Type:   codec.Error,
				}, nil)

				// if the server request is an EOS error we let the socket know
				// sometimes the socket is already closed on the other side, so we can ignore that error
				alreadyClosed := serveRequestError == lastStreamResponseError && writeError == io.EOF

				// could not write error response
				if writeError != nil && !alreadyClosed {
					log.Logf("rpc: unable to write error response: %v", writeError)
				}
			}

			mtx.Lock()
			delete(sockets, id)
			mtx.Unlock()

			// signal we're done
			if s.wg != nil {
				s.wg.Done()
			}

			// done with this socket
			wg.Done()
		}(id, psock)
	}
}

func (s *rpcServer) newCodec(contentType string) (codec.NewCodec, error) {
	if cf, ok := s.opts.Codecs[contentType]; ok {
		return cf, nil
	}
	if cf, ok := DefaultCodecs[contentType]; ok {
		return cf, nil
	}
	return nil, fmt.Errorf("Unsupported Content-Type: %s", contentType)
}

func (s *rpcServer) Options() Options {
	s.RLock()
	opts := s.opts
	s.RUnlock()
	return opts
}

func (s *rpcServer) Init(opts ...Option) error {
	s.Lock()
	for _, opt := range opts {
		opt(&s.opts)
	}

	// update router if its the default
	if s.opts.Router == nil {
		r := newRpcRouter()
		r.hdlrWrappers = s.opts.HdlrWrappers
		r.serviceMap = s.router.serviceMap
		s.router = r
	}

	s.Unlock()
	return nil
}

func (s *rpcServer) NewHandler(h interface{}, opts ...HandlerOption) Handler {
	return s.router.NewHandler(h, opts...)
}

func (s *rpcServer) Handle(h Handler) error {
	s.Lock()
	defer s.Unlock()

	if err := s.router.Handle(h); err != nil {
		return err
	}

	s.handlers[h.Name()] = h

	return nil
}

func (s *rpcServer) NewSubscriber(topic string, sb interface{}, opts ...SubscriberOption) Subscriber {
	return newSubscriber(topic, sb, opts...)
}

func (s *rpcServer) Subscribe(sb Subscriber) error {
	sub, ok := sb.(*subscriber)
	if !ok {
		return fmt.Errorf("invalid subscriber: expected *subscriber")
	}
	if len(sub.handlers) == 0 {
		return fmt.Errorf("invalid subscriber: no handler functions")
	}

	if err := validateSubscriber(sb); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()
	_, ok = s.subscribers[sub]
	if ok {
		return fmt.Errorf("subscriber %v already exists", s)
	}
	s.subscribers[sub] = nil
	return nil
}

func (s *rpcServer) Register() error {
	var err error
	var advt, host, port string

	// parse address for host, port
	config := s.Options()

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
	}

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		// ipv6 address in format [host]:port or ipv4 host:port
		host, port, err = net.SplitHostPort(advt)
		if err != nil {
			return err
		}
	} else {
		host = advt
	}

	addr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	// make copy of metadata
	md := make(metadata.Metadata)
	for k, v := range config.Metadata {
		md[k] = v
	}

	// mq-rpc(eg. nats) doesn't need the port. its addr is queue name.
	if port != "" {
		addr = mnet.HostPort(addr, port)
	}

	// register service
	node := &registry.Node{
		Id:       config.Name + "-" + config.Id,
		Address:  addr,
		Metadata: md,
	}

	node.Metadata["transport"] = config.Transport.String()
	node.Metadata["broker"] = config.Broker.String()
	node.Metadata["server"] = s.String()
	node.Metadata["registry"] = config.Registry.String()
	node.Metadata["protocol"] = "mucp"

	s.RLock()
	// Maps are ordered randomly, sort the keys for consistency
	var handlerList []string
	for n, e := range s.handlers {
		// Only advertise non internal handlers
		if !e.Options().Internal {
			handlerList = append(handlerList, n)
		}
	}
	sort.Strings(handlerList)

	var subscriberList []*subscriber
	for e := range s.subscribers {
		// Only advertise non internal subscribers
		if !e.Options().Internal {
			subscriberList = append(subscriberList, e)
		}
	}
	sort.Slice(subscriberList, func(i, j int) bool {
		return subscriberList[i].topic > subscriberList[j].topic
	})

	var endpoints []*registry.Endpoint
	for _, n := range handlerList {
		endpoints = append(endpoints, s.handlers[n].Endpoints()...)
	}
	for _, e := range subscriberList {
		endpoints = append(endpoints, e.Endpoints()...)
	}
	s.RUnlock()

	service := &registry.Service{
		Name:      config.Name,
		Version:   config.Version,
		Nodes:     []*registry.Node{node},
		Endpoints: endpoints,
	}

	s.Lock()
	registered := s.registered
	s.Unlock()

	if !registered {
		log.Logf("Registry [%s] Registering node: %s", config.Registry.String(), node.Id)
	}

	// create registry options
	rOpts := []registry.RegisterOption{registry.RegisterTTL(config.RegisterTTL)}

	if err := config.Registry.Register(service, rOpts...); err != nil {
		return err
	}

	// already registered? don't need to register subscribers
	if registered {
		return nil
	}

	s.Lock()
	defer s.Unlock()

	s.registered = true

	for sb, _ := range s.subscribers {
		handler := s.createSubHandler(sb, s.opts)
		var opts []broker.SubscribeOption
		if queue := sb.Options().Queue; len(queue) > 0 {
			opts = append(opts, broker.Queue(queue))
		}

		if cx := sb.Options().Context; cx != nil {
			opts = append(opts, broker.SubscribeContext(cx))
		}

		if !sb.Options().AutoAck {
			opts = append(opts, broker.DisableAutoAck())
		}

		sub, err := config.Broker.Subscribe(sb.Topic(), handler, opts...)
		if err != nil {
			return err
		}
		log.Logf("Subscribing %s to topic: %s", node.Id, sub.Topic())
		s.subscribers[sb] = []broker.Subscriber{sub}
	}

	return nil
}

func (s *rpcServer) Deregister() error {
	var err error
	var advt, host, port string

	config := s.Options()

	// check the advertise address first
	// if it exists then use it, otherwise
	// use the address
	if len(config.Advertise) > 0 {
		advt = config.Advertise
	} else {
		advt = config.Address
	}

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		// ipv6 address in format [host]:port or ipv4 host:port
		host, port, err = net.SplitHostPort(advt)
		if err != nil {
			return err
		}
	} else {
		host = advt
	}

	addr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	// mq-rpc(eg. nats) doesn't need the port. its addr is queue name.
	if port != "" {
		addr = mnet.HostPort(addr, port)
	}

	node := &registry.Node{
		Id:      config.Name + "-" + config.Id,
		Address: addr,
	}

	service := &registry.Service{
		Name:    config.Name,
		Version: config.Version,
		Nodes:   []*registry.Node{node},
	}

	log.Logf("Registry [%s] Deregistering node: %s", config.Registry.String(), node.Id)
	if err := config.Registry.Deregister(service); err != nil {
		return err
	}

	s.Lock()

	if !s.registered {
		s.Unlock()
		return nil
	}

	s.registered = false

	for sb, subs := range s.subscribers {
		for _, sub := range subs {
			log.Logf("Unsubscribing %s from topic: %s", node.Id, sub.Topic())
			sub.Unsubscribe()
		}
		s.subscribers[sb] = nil
	}

	s.Unlock()
	return nil
}

func (s *rpcServer) Start() error {
	s.RLock()
	if s.started {
		s.RUnlock()
		return nil
	}
	s.RUnlock()

	config := s.Options()

	// start listening on the transport
	ts, err := config.Transport.Listen(config.Address)
	if err != nil {
		return err
	}

	log.Logf("Transport [%s] Listening on %s", config.Transport.String(), ts.Addr())

	// swap address
	s.Lock()
	addr := s.opts.Address
	s.opts.Address = ts.Addr()
	s.Unlock()

	// connect to the broker
	if err := config.Broker.Connect(); err != nil {
		return err
	}

	bname := config.Broker.String()

	log.Logf("Broker [%s] Connected to %s", bname, config.Broker.Address())

	// use RegisterCheck func before register
	if err = s.opts.RegisterCheck(s.opts.Context); err != nil {
		log.Logf("Server %s-%s register check error: %s", config.Name, config.Id, err)
	} else {
		// announce self to the world
		if err = s.Register(); err != nil {
			log.Logf("Server %s-%s register error: %s", config.Name, config.Id, err)
		}
	}

	exit := make(chan bool)

	go func() {
		for {
			// listen for connections
			err := ts.Accept(s.ServeConn)

			// TODO: listen for messages
			// msg := broker.Exchange(service).Consume()

			select {
			// check if we're supposed to exit
			case <-exit:
				return
			// check the error and backoff
			default:
				if err != nil {
					log.Logf("Accept error: %v", err)
					time.Sleep(time.Second)
					continue
				}
			}

			// no error just exit
			return
		}
	}()

	go func() {
		t := new(time.Ticker)

		// only process if it exists
		if s.opts.RegisterInterval > time.Duration(0) {
			// new ticker
			t = time.NewTicker(s.opts.RegisterInterval)
		}

		// return error chan
		var ch chan error

	Loop:
		for {
			select {
			// register self on interval
			case <-t.C:
				s.RLock()
				registered := s.registered
				s.RUnlock()
				if err = s.opts.RegisterCheck(s.opts.Context); err != nil && registered {
					log.Logf("Server %s-%s register check error: %s, deregister it", config.Name, config.Id, err)
					// deregister self in case of error
					if err := s.Deregister(); err != nil {
						log.Logf("Server %s-%s deregister error: %s", config.Name, config.Id, err)
					}
				} else {
					if err := s.Register(); err != nil {
						log.Logf("Server %s-%s register error: %s", config.Name, config.Id, err)
					}
				}
			// wait for exit
			case ch = <-s.exit:
				t.Stop()
				close(exit)
				break Loop
			}
		}

		// deregister self
		if err := s.Deregister(); err != nil {
			log.Logf("Server %s-%s deregister error: %s", config.Name, config.Id, err)
		}

		// wait for requests to finish
		if s.wg != nil {
			s.wg.Wait()
		}

		// close transport listener
		ch <- ts.Close()

		// disconnect the broker
		config.Broker.Disconnect()

		// swap back address
		s.Lock()
		s.opts.Address = addr
		s.Unlock()
	}()

	// mark the server as started
	s.Lock()
	s.started = true
	s.Unlock()

	return nil
}

func (s *rpcServer) Stop() error {
	s.RLock()
	if !s.started {
		s.RUnlock()
		return nil
	}
	s.RUnlock()

	ch := make(chan error)
	s.exit <- ch

	var err error
	select {
	case err = <-ch:
		s.Lock()
		s.started = false
		s.Unlock()
	}

	return err
}

func (s *rpcServer) String() string {
	return "mucp"
}

