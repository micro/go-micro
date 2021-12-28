# 4.5.0 (2021/12/19)

### Bug Fixes

- nats deregister issue (#2384)
- fixing f.IsExported undefined issue (#2382)
- fix http_transport Recv and Close race condition on buff (#2374)
- update protoc-gen-micro install doc
- zookeeper registry delete event (#2372)
- delete redundant lines (#2364)
- modify the dependencies urls (#2363)
- ignore unexported field (#2354)
- Fix Micro CLI's proto comments (#2353)

### Features

- Extend client mock with ability to test publish, and a few useful method like SetResponse and SetSubscriber (#2375)
- go micro dashboard (#2361)

# 4.4.0 (2021/11/11)

### Bug Fixes

- fix(#2333): etcd grpc marshal issue (#2334)

### Features

- upgrade to go 1.17 (#2346)
- add nats and redis events plugins
- add events package (#2341)

# 4.3.0 (2021/11/01)

### Bug Fixes

- flatten cli (#2332)
- m3o client changed
- use vanity url for cli command
- fix broker nsq plugin nil pointer error (#2329)
- fix config json slice parsing (#2330)
- replace ioutil with io and os (#2327)
- fixing #2308 (#2326)
- Fix Micro CLI by removing replace line in go.mod
- remove unnecessary dependencies between plugins
- 1. use default memory registry in grpc plugins (#2317)
- Add update rule to Makefile (#2315)
- Plugins (#2311)

### Features

- Rename gomu to micro (#2325)
- stream CloseSend (#2323)
- strip protoc-gen-micro go mod

### BREAKING CHANGES

- go install go-micro.dev/v4/cmd/micro@v4
- go install go-micro.dev/v4/cmd/protoc-gen-micro@v4
- upgrade go micro to support stream.CloseSend

# 4.2.1 (2021/10/14)

### Bug Fixes

- fix gomu.

# 4.2.0 (2021/10/13)

### Bug Fixes

- fix examples go mod.
- update go sums.
- move to go-micro.dev.
- upgrade protoc-gen-micro to v4.

# 4.1.0 (2021/10/12)

- v4.1.0.

# 4.0.0 (2021/10/12)

### Features

- Vanity URL go-micro.dev([#2271](https://github.com/asim/go-micro/issues/2271)).

### BREAKING CHANGES

- upgrade github.com/asim/go-micro/v3 to go-micro.dev/v4.

# 3.7.0 (2021/10/11)

- Add latest version (#2303).

# 3.6.0 (2021/08/23)

- Minor fixes https://github.com/asim/go-micro/compare/v3.5.2...c7195aae9817db4eaf5483990fcb8706f86d3002.

# 3.5.2 (2021/07/06)

- Tag it and bag it.

# 3.5.1 (2021/04/20)

- Minor bug fixes.

# 3.5.0 (2021/01/29)

- kill the bugs.

# 3.0.1 (2021/01/20)

- Tag with protoc changes.

# 3.0.0 (2021/01/20)

- V3.

### BREAKING CHANGES

- upgrade github.com/micro/go-micro/v2 to github.com/asim/go-micro/v3.
- change default transport from gRPC to mucp, using grpc server/client plugins.

# 3.0.0-beta.3 (2020/09/29)

- Secret implementation of config. Supporting config merge (#2027)
- remove transport options
- read service package (#2026)
- env config implementation (#2024)
- runtime: remove builder package (moved to micro) (#2023)
- Fix branch names support for k8s runtime (#2020)
- fix config bug (#2021)
- runtime: minor fixes for local runtime (#2019)
- remove memcache and update gomod
- Add errors to config methods (#2015)
- store/file: fix segmentation violation bug (#2013)
- Config interface change (#2010)
- proxy/grpc: fix client streaming bug (EOF not sent to the server) (#2011)
- client/grpc: fix stream closed bug (#2009)
- store/file: don't keep connection to boltdb open (#2006)
- runtime/builder with golang implementation (#2003)
- store: implement s3 blob store (#2005)
- store: add blob interface with file implementation (#2004)
- auth: remove micro specific code (#1999)
- Fix running subfolders (#1998)
- api: fix request body re-sequencing bug (#1996)
- add Name to auth.Account as a user friendly alias (#1992)
- Fixing top level run outside repo (#1993)
- runtime: normalised service statuses (#1994)
- Add 'Namespace' header to allowed CORS headers (#1990)
- Remove all the external plugins except grpc (#1988)
- util/kubernetes: fix TCPSocketAction bug (#1987)
- Fixing the metric tagging issue here (#1986).

# 3.0.0-beta.2 (2020/09/05)

- Cut a v3 beta 2.

# 3.0.0-beta (2020/08/12)

- write nil when expiry is zero.

# 3.0.0-alpha (2020/07/27)

- v3 refactor (#1868).

# 2.9.1 (2020/07/03)

- push tags to docker hub (#1766).

# 2.9.0 (2020/06/12)

- Fix regex detection. Fixes #1663 (#1696).

# 2.9.0-rc5 (2020/06/11)

- Merge branch 'master' into release-2.9.0.

# 2.9.0-rc4 (2020/06/11)

- Merge branch 'master' into release-2.9.0.

# 2.9.0-rc1 (2020/06/11)

- Merge branch 'master' into release-2.9.0.

# 2.8.0 (2020/05/31)

- Rewrite Auth interface to use Rules
- Add Cache interface into the Client for request caching
- Fix atomic sequence updates in Client
- Update go mod deps
- Fix ipv6 parsing in mdns registry
- Add namespacing to the default runtime
- Replace go-git with v5
- Increase register ttl to 90 seconds.

# 2.7.0 (2020/05/18)

- Fix the rpc handler json rpc body parsing
- Use caddyserver/certmagic instead of mholt
- Add HasRole to Account
- Add jwt refresh token generation
- Fix rpc stream close locking race
- Add auth namespace env var
- Strip the router penalty code
- Add file upload util
- Fix killing processes in runtime
- Pass namespace to runtime commands
- Generate account on start
- Check errors in cockroachdb.

# 2.6.0 (2020/05/04)

- Fix discord bot authentication header
- Improve api rpc regexp matching
- Change auth account access via context
- Create a jwt implementation of auth
- Fix grpc content-type encoding bug
- Consolidate proxy/network env var logic
- Change secrets interface naming
- Log file path in the logger
- Change location of network resolver
- Add Store to service options
- Fix default runtime log parsing
- Add namespace checks to k8s runtime
- Add proper git checkout in local runtime
- Add database/table options for store
- Add pki implementation
- Import qson.

# 2.5.0 (2020/04/15)

- api/router/registry: extract path based parameters from url to req (#1530).

# 2.4.0 (2020/03/31)

- There can be only one! (#1445).

# 2.3.0 (2020/03/17)

- grpc client/server fixes (#1355).

# 2.2.0 (2020/02/28)

- Rename Auth Validate to Verify
- Replaces noop auth with base32 generated tokens
- Change Excludes to Exclude
- Add token option to auth
- Add profile option and flags for debug
- Add config loading for auth token
- Move before start to before listening.

# 2.1.2 (2020/02/24)

- fix router panic (#1254).

# 2.1.1 (2020/02/23)

- update go modules (#1240).

# 2.1.0 (2020/02/13)

- Exclude Stats & Trace from Auth (#1192).

# 2.0.0 (2020/01/30)

- v2 release.

# 1.18.0 (2019/12/08)

- Add golang ci linter
- Add race detection to travis
- Please the linter
- Do some perf optimisations on slice alloc
- Move http broker to use single entry in registry
- Strip the grpc metadata filtering
- Strip the old codec usage
- Disable retries in client when MICRO_PROXY is enabled
- Strip old X-Micro headers
- Add debug/log streaming implementations
- Add first debug/log interface
- Huge network/tunnel refactor to fix bugs
- Fix proxy slice allocation bug
- Splay out some of the network events
- Default to AdvertiseLocal for router
- Add runtime filtering with Type
- Remove SIGKILL processing.

# 1.17.1 (2019/11/27)

- fix rpc server go routine leak
- add a psuedo socket pool
- update debug buffer to return entries.

# 1.17.0 (2019/11/27)

- Add github related issue templates
- Add Dockerfile for predownloaded go-micro source
- Regenerate all the protos to move to \*.pb.micro.go
- Fix api handler to parse text/plain as default content type
- Fix event handler to allow GET requests
- Change http broker ids to go.micro.http.broker-uuid
- Require protocol field in metadata to query services via client
- Process raw frames in call to Publish
- Complete proxy support for processing messages
- Proxy support for publishing of messages
- Fix grpc connection leak by always closing the connection
- Add a debug ring buffer
- Add broker to tunnel and network
- Force network dns resolver to use cloudflare 1.0.0.1
- Add option to specify whether server should handle signalling
- Change mdns request timeout to 10ms rather than 100ms
- Add router AdvertiseNone and AdvertiseLocal strategies
- Rename runtime packager to builder
- Add full support for a kubernetes runtime.

# 1.16.0 (2019/11/09)

- Pre-make slices for perf optimisation
- Add runtime flag and k8s runtime
- Add debug/profile for pprof profiling
- Reduce go routines in mdns registry and registry cache
- Optimise the router flap detection.

# 1.15.1 (2019/11/03)

- Router recovery penalty should be below 500.

# 1.15.0 (2019/11/03)

- go fmt -s
- web generate service on registration
- downgrade some network messages to trace
- fix tunnel panic on deleting link
- add postgres store
- change grpc recover logging
- add runtime service
- add kubernetes runtime
- add runtime notifier
- proxy add header based routing for Micro-{Gateway, Router, Network}
- network hash address based on service + node id
- metadata add mergecontext function.

# 1.14.0 (2019/10/25)

- Remove consul registry
- Change store Sync endpoint to List
- Remove cloudflare-go usage in store
- Add non-backwards compatible link changes.

# 1.13.2 (2019/10/22)

- Fix proxy selection to use round robin strategy.

# 1.13.1 (2019/10/19)

- Fix divide by zero bug in broker.

# 1.13.0 (2019/10/18)

- Fix network recursive read lock bug
- Add certmagic random pull time
- Strip http broker topic: prefix.

# 1.12.0 (2019/10/17)

- Add ACME Provider interface
- Implement certmagic ACME Provider
- Add certmagic Store implementation
- Add broker service implementation
- Add ability to set grpc dial and call options
- Add etcd registry and other plugins
- Add Network.Connect rpc endpoint
- Resolve network node dns names
- Support Network.Routes querying
- Fix caching registry bugs
- Move gossip registry to go-plugins
- Add router advertise strategy
- Add Cloudflare store implementation
- Add store service implementation.

# 1.11.3 (2019/10/12)

- Fix the quic-go checksum mismatch by updating to 0.12.1.

# 1.11.2 (2019/10/12)

- Fix cache error check.

# 1.11.1 (2019/10/07)

- Fix cache registry deadlocking bug.

# 1.11.0 (2019/10/01)

- This is likely the last release of v1.

# 1.10.0 (2019/09/11)

- Add grpc client code application/grpc content-type
- Move client to use stream dialer
- Add network implementation
- Add dynamic plugin loading
- Add multilink usage in proxy
- Add registry implementation
- Scope mdns to .micro domain
- Support grpc server processing by default
- Add tunnel broker.

# 1.9.1 (2019/08/19)

- Fix waitgroup race condition.

# 1.9.0 (2019/08/19)

- Fix grpc codec for broker publishing
- Use the connection pool for streaming
- Send EOS from client when closing stream
- Add stream header to mucp protocol
- Add stream multiplexing in the server
- Fix watcher bug in file config source
- Fix monitoring watcher to only look at mucp services
- Only check router status on lookup failure
- Fix proxy streaming and client request processing
- Fix host:port processing for messaging systems
- Add start method to the router
- Fix router race condition for default values
- Add loopback detection to the tunnel
- Add connection retry logic to tunnel
- Make log levels accessible for the logger
- Add proxy muxer for internal calls.

# 1.8.3 (2019/08/12)

- Fix nats draining
- More verbose selector errors to return service name
- Move handler debug package
- Add a monitoring package
- Fix consul address parsing
- Fix server extraction code
- Add tunnel implementation
- Add util log level
- Add util io package to wrap transport socket.

# 1.8.2 (2019/08/06)

- Point release for micro
- Adds travis caching
- Removes unused network code
- Adds tunnel interface
- Consul agent check
- Router handler interface
- Non host:port fixes.

# 1.8.1 (2019/07/31)

- Use mdns 0.2.0 release tag.

# 1.8.0 (2019/07/29)

- Move the selector into client
- Change broker.Publication to broker.Event
- Move cmd into config
- Enable default json processing in api
- Remove port from registry
- Memory broker/transport race fixes
- GRPC codec fix
- Client pool interface
- Router interface/service implementations
- Config decoding fixes
- Memory store expiration fix
- Network link/tunnel/resolver packages
- Proxy router caching
- Registry util functions.

# 1.7.0 (2019/06/21)

- Update go mod
- Move mock data out of memory registry
- wrap the grpc codecs to support framing
- change grpc resolution to use service.method
- support full proxying via grpc
- add text codec
- move data/store
- add network interface
- add router package and implementation
- move options to config/options
- send gossip updates on register/deregister
- fix node add/del bug
- add handler wrapper back into core router.

# 1.6.0 (2019/06/07)

- Massive go.mod dependency cleanup _ Moved etcd, memcache, redis sync things to go-plugins _ uuid to google uuid \* blew away go.mod
- Add better proxy interface and features
- Add new options interface.

# 1.5.0 (2019/06/05)

- Fix go mod issues.

# 1.4.0 (2019/06/04)

- Final consolidation of all libraries.

# 1.3.1 (2019/06/03)

- Fix broken pipe bug. Don't send message when client closed connection..

# 1.3.0 (2019/05/31)

- The great rewrite.

# 1.2.0 (2019/05/22)

- Update go mod
- Fix mock client
- Fix retries logic
- Fix consul api change
- Use consul client for watcher
- Fix gossip data races
- Add registry check function.

# 1.1.0 (2019/03/28)

- Update go mod
- Fix endpoint extractor generation.

# 1.0.0 (2019/03/05)

- 1.0.0 release.

# 0.27.1 (2019/03/05)

- Fix nil consul client.

# 0.27.0 (2019/02/23)

- Remove buff check in http transport
- Change default version to latest
- Add exchange routing
- Update go modules.

# 0.26.1 (2019/02/13)

- Fix gossip registry
- Update go modules for rcache.

# 0.26.0 (2019/02/13)

- Update go modules
- Add gossip registry rejoin
- Move selector to rcache.

# 0.25.0 (2019/02/04)

- Add server request body.

# 0.24.1 (2019/02/01)

- Various bug fixes
- Backwards compatible with 0.14 and older
- Fix mdns and gossip race conditions
- Use official h2c server
- Enable support for MICRO_PROXY.

# 0.24.0 (2019/01/30)

- Add go mod.

# 0.23.0 (2019/01/29)

- Move headers from X-Micro to Micro
- Remove Register/Deregister methods from server
- Move register_interval to be internal
- Add subscriber context option.

# 0.22.1 (2019/01/22)

- Fix broken error handling
- now returns error from ServeRequest router.

# 0.22.0 (2019/01/18)

- Address backwards compatibility.

# 0.21.0 (2019/01/17)

- Make MDNS the default registry
- Move mocks to be memory implementations
- Add metadata.Copy function.

# 0.20.0 (2019/01/14)

- BREAKING CHANGES.

# 0.17.0 (2019/01/03)

- Offline inbox for http broker
- JSON/Proto/GRPC codecs
- HTTP proxy from environment.

# 0.16.0 (2018/12/29)

- Fix cache/gossip data race
- Rename cache selector to registry.

# 0.15.1 (2018/12/18)

- Selector cache lookup optimization.

# 0.15.0 (2018/12/13)

- Public NewSubscribeOptions
- http2 broker support
- Timeout error function
- Consul Query Options
- Gossip registry
- RPC Codec renaming.

# 0.14.1 (2018/11/22)

- bug fix socket headers.

# 0.14.0 (2018/11/21)

- use google uuid
- add http handler option.

# 0.13.0 (2018/11/15)

- add local/remote ip methods
- various linting things
- get checks on 0 ttl
- accept loop.

# 0.12.0 (2018/10/09)

- reorder server flag
- atomic increment sequence
- new error method.

# 0.11.0 (2018/08/24)

- Support Consul Connect registration
- Add/Use Init for initialisation from cmd.

# 0.10.0 (2018/07/26)

- Fix broker locking
- Add RetryOnError as default retry policy
- Fix mock client reflection
- Support dialtimeout only above 0
- Add verbose client errors
- Allow client retries to be 0.

# 0.9.0 (2018/06/09)

- Reset server address on shutdown
- Set default pool size to 1
- Support reinitialising connection pool
- Set retries to 1 by default
- Return error for subscribers.

# 0.8.0 (2018/04/20)

- Rework of interfaces.

# 0.7.0 (2018/04/10)

- Move misc to util package
- Add register ttl and interval flags
- Fix protoc-gen-micro example.

# 0.6.0 (2018/04/05)

- Add consul TCP check
- Atomic increment rpc stream sequence.

# 0.5.0 (2018/03/04)

- Support consul services without version
- Switch to stdlib context.

# 0.4.0 (2018/02/19)

- Add WatchOption which allows filtering by service
- Add Options method to registry
- Add Conflict error
- Only watch selected services in cache.

# 0.3.0 (2018/01/02)

- https support for consul
- subscriber deadlock fix
- selector top level option.

# 0.2.0 (2017/10/29)

- Performance improvements.

# 0.1.4 (2017/09/04)

- sort handler/subscriber endpoints
- pass options to new subscriber.

# 0.1.3 (2017/08/15)

- Bug fix nil consul http client.

# 0.1.2 (2017/07/20)

- respond when codec errors out.

# 0.1.1 (2017/06/12)

- Fix potential panic/waitgroup bug.

# 0.1.0 (2017/06/12)

- Initial release.
