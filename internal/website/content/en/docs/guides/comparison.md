---
layout: default
---

# Framework Comparison

How Go Micro compares to other Go microservices frameworks.

## Quick Comparison

| Feature | Go Micro | go-kit | gRPC | Dapr |
|---------|----------|--------|------|------|
| **Learning Curve** | Low | High | Medium | Medium |
| **Boilerplate** | Low | High | Medium | Low |
| **Plugin System** | Built-in | External | Limited | Sidecar |
| **Service Discovery** | Yes (mDNS, Consul, etc) | No (BYO) | No | Yes |
| **Load Balancing** | Client-side | No | No | Sidecar |
| **Pub/Sub** | Yes | No | No | Yes |
| **Transport** | HTTP, gRPC, NATS | BYO | gRPC only | HTTP, gRPC |
| **Zero-config Dev** | Yes (mDNS) | No | No | No (needs sidecar) |
| **Production Ready** | Yes | Yes | Yes | Yes |
| **Language** | Go only | Go only | Multi-language | Multi-language |

## vs go-kit

### go-kit Philosophy
- "Just a toolkit" - minimal opinions
- Compose your own framework
- Maximum flexibility
- Requires more decisions upfront

### Go Micro Philosophy
- "Batteries included" - opinionated defaults
- Swap components as needed
- Progressive complexity
- Get started fast, customize later

### When to Choose go-kit
- You want complete control over architecture
- You have strong opinions about structure
- You're building a custom framework
- You prefer explicit over implicit

### When to Choose Go Micro
- You want to start coding immediately
- You prefer conventions over decisions
- You want built-in service discovery
- You need pub/sub messaging

### Code Comparison

**go-kit** (requires more setup):
```go
// Define service interface
type MyService interface {
    DoThing(ctx context.Context, input string) (string, error)
}

// Implement service
type myService struct{}

func (s *myService) DoThing(ctx context.Context, input string) (string, error) {
    return "result", nil
}

// Create endpoints
func makeDo ThingEndpoint(svc MyService) endpoint.Endpoint {
    return func(ctx context.Context, request interface{}) (interface{}, error) {
        req := request.(doThingRequest)
        result, err := svc.DoThing(ctx, req.Input)
        if err != nil {
            return doThingResponse{Err: err}, nil
        }
        return doThingResponse{Result: result}, nil
    }
}

// Create transport (HTTP, gRPC, etc)
// ... more boilerplate ...
```

**Go Micro** (simpler):
```go
type MyService struct{}

type Request struct {
    Input string `json:"input"`
}

type Response struct {
    Result string `json:"result"`
}

func (s *MyService) DoThing(ctx context.Context, req *Request, rsp *Response) error {
    rsp.Result = "result"
    return nil
}

func main() {
    svc := micro.NewService("myservice")
    svc.Init()
    svc.Handle(new(MyService))
    svc.Run()
}
```

## vs gRPC

### gRPC Focus
- High-performance RPC
- Multi-language support via protobuf
- HTTP/2 transport
- Streaming built-in

### Go Micro Scope
- Full microservices framework
- Service discovery
- Multiple transports (including gRPC)
- Pub/sub messaging
- Pluggable components

### When to Choose gRPC
- You need multi-language services
- Performance is critical
- You want industry-standard protocol
- You're okay managing service discovery separately

### When to Choose Go Micro
- You need more than just RPC (pub/sub, discovery, etc)
- You want flexibility in transport
- You're building Go-only services
- You want integrated tooling

### Integration

You can use gRPC with Go Micro for native gRPC compatibility:
```go
import (
    grpcServer "go-micro.dev/v6/server/grpc"
    grpcClient "go-micro.dev/v6/client/grpc"
)

svc := micro.NewService("myservice",
    micro.Server(grpcServer.NewServer()),
    micro.Client(grpcClient.NewClient()),
)
```

See [Native gRPC Compatibility](grpc-compatibility.md) for a complete guide.

## vs Dapr

[Dapr](https://dapr.io/) is a distributed application runtime. Its building
blocks cover service invocation, state, pub/sub, bindings, secrets,
configuration, distributed locks, actors, jobs, and workflow, usually accessed
through a sidecar from many languages. [Dapr Agents](https://docs.dapr.io/developing-ai/dapr-agents/)
adds an agent framework on top of those runtime capabilities.

Go Micro overlaps with Dapr on distributed-systems primitives, but the product
shape is different: Go Micro is a Go framework where services, agents, tools,
and flows are built from the same runtime. A service endpoint can become an
AI-callable tool, and an agent is itself a registered service with memory,
guardrails, planning, delegation, MCP, and A2A around it.

### Decision table

| Need | Prefer Go Micro | Prefer Dapr | Use both |
|---|---|---|---|
| **Primary language** | Your core runtime is Go and you want library-native APIs | You run a polyglot estate and want one sidecar API across languages | Go services use Go Micro while non-Go services expose Dapr APIs |
| **Agent model** | Agents should be ordinary services: registered, discoverable, callable by RPC, MCP, and A2A | Agents are primarily Python applications using Dapr Agents | Dapr-hosted agents call Go Micro MCP tools, or Go Micro agents call Dapr-backed services |
| **Tools** | Existing service endpoints should become tools with minimal extra code | Tools are modeled through Dapr components, bindings, or agent framework code | Use Dapr components behind Go Micro services that expose a stable tool surface |
| **Workflows** | Deterministic steps should live beside Go services and agents in the same codebase | You want Dapr Workflow's sidecar-backed orchestration model across languages | Let Dapr own cross-language workflows and let Go Micro own Go-native agent/tool execution |
| **State and pub/sub** | You want Go interfaces and pluggable packages directly in-process | You want component YAML and sidecar portability across backing services | Put portable infrastructure behind Dapr and domain/tool logic in Go Micro |
| **Deployment** | You want a simple Go binary/runtime first, with Kubernetes support as an explicit deployment target | You are already standardized on Dapr sidecars in Kubernetes | Run Go Micro services in clusters that already have Dapr for shared infrastructure |
| **Interop** | MCP and A2A are first-class requirements for exposing services and agents | Dapr's app APIs and agent framework are the integration boundary | Bridge through MCP/A2A at the agent edge and Dapr APIs at the infrastructure edge |

### When to choose Dapr

- You need a **polyglot** runtime contract for Node, Python, Java, .NET, Go, and
  other services.
- Your platform team already operates sidecars and component configuration across
  Kubernetes clusters.
- You want Dapr's standard building blocks for state, pub/sub, bindings, secrets,
  actors, jobs, and workflow more than you want a Go-native service framework.
- You are adopting Dapr Agents and want to stay in its Python-first agent stack.

### When to choose Go Micro

- You are building mostly in Go and want the agent harness to be the same runtime
  as your services.
- You want service methods and their comments/examples to become AI-callable tools
  without maintaining a separate tool layer.
- You want agents to be deployed, discovered, called, load-balanced, and inspected
  like ordinary services.
- You need MCP and A2A at the agent/service boundary, not only an internal
  application API.
- You prefer library-native composition and direct Go interfaces over sidecar
  component wiring.

### Where Go Micro still needs to prove itself

Dapr has a mature platform narrative and broad deployment footprint. Go Micro's
agent-harness story is sharper for Go teams, but production adoption depends on
keeping the no-secret getting-started path green, documenting durability
semantics clearly, proving MCP/A2A conformance with external clients, and making
Kubernetes deployment first-class.

### Practical migration path

1. Start with one Go Micro service that wraps a real domain capability.
2. Add doc comments and examples so the endpoint is useful as an agent tool.
3. Expose it through MCP for external agents or through A2A if the capability is
   itself an agent.
4. If your platform already uses Dapr, keep Dapr components behind the service
   boundary and let Go Micro present the agent/tool contract.
5. Move deterministic multi-step work into flows only after the service/tool
   boundary is stable.

## vs Agent Frameworks (Google ADK)

[ADK](https://adk.dev/) (Agent Development Kit) is Google's open-source, code-first
framework for building AI agents. It spans several languages (Python, TypeScript,
Go, Java, Kotlin); [`adk-go`](https://github.com/google/adk-go) is the Go
implementation. It's model-agnostic (optimized for Gemini), speaks MCP and A2A,
and supports multi-agent systems, evaluation, and deployment to Cloud Run / GKE.

They overlap on agents but solve different problems. ADK is a library for building
an agent process — you define an agent, its tools, and a model, then run and deploy
it. Go Micro is the harness around agents once they operate real systems: service
discovery, inter-service RPC, pub/sub, durable flows, tool execution, and deployment.
Those pieces are out of scope for ADK, and you bring your own.

In Go Micro an agent is built as an ordinary service: it registers in the registry,
is callable by RPC (`Agent.Chat`) and over A2A, and other services and agents
discover and call it the same way they call anything else. Its endpoints are exposed
as MCP tools automatically. So once you have more than one agent or service, Go Micro
also gives you the discovery, RPC, pub/sub, config, and deployment around them.

| | Go Micro | Google ADK |
|---|----------|------------|
| **Primary unit** | A harnessed service (an agent is a service with an LLM inside) | An agent |
| **Service discovery / registry** | Built-in (mDNS, Consul, etcd) | Not in scope |
| **Inter-service RPC, load balancing, pub/sub** | Built-in | Not in scope |
| **MCP** | Every service endpoint is automatically an MCP tool (no extra code) | MCP tools, wired explicitly |
| **A2A** | Agents are A2A-reachable services | Supported |
| **Deterministic orchestration** | Flows | Graph workflows |
| **Multi-agent** | Agents discover & call each other via the registry; `plan`/`delegate` built in | Composition, routing, workflow patterns |
| **Evaluation suite** | Harnesses/conformance today; first-class evaluation is a gap | Yes (criteria, user/env simulation, metrics) |
| **Context engineering** | Store-backed memory | "Context as source code" (auto filter/summarize/token tracking) |
| **Languages** | Go | Python, TypeScript, Go, Java, Kotlin |
| **Backing** | Community | Google |

### When to choose ADK
- You want an agent framework with first-class **evaluation** and context tooling
- You're polyglot, or invested in the Google Cloud / Gemini ecosystem
- You want a cross-language A2A ecosystem with Google's backing

### When to choose Go Micro
- You want an **agent harness** where agents and services are the same thing —
  registered, discoverable, load-balanced, and deployed the same way
- You want your existing services to become agent tools with **zero extra code**
  (every endpoint is an MCP tool automatically)
- You're building in Go and want one set of primitives for services, agents, and flows

### They interoperate

Both speak **MCP** and **A2A**, so this isn't strictly either/or: a Go Micro agent
and an ADK agent (in any language) can call each other over A2A, and either can
consume the other's MCP tools. A common pattern is to run Go Micro as the service
mesh / runtime and let ADK (or any A2A agent) plug into it.

## vs tRPC-Agent-Go

[tRPC-Agent-Go](https://github.com/trpc-group/trpc-agent-go) (maintained by tRPC-Group,
validated inside Tencent) is a production-grade Go framework for agent systems:
LLM / Chain / Parallel / Cycle / Graph agents, function tools, MCP, A2A, AG-UI, Redis
memory and RAG, evaluation, agent self-evolution, and OpenTelemetry. It's a serious,
well-resourced project.

They overlap heavily on agents but take a different approach. tRPC-Agent-Go is an **agent
SDK you run alongside your services** — you compose agents and tools into graphs and
conditional workflows, and your microservices (tRPC) live separately and are called
into. Go Micro starts from the premise that **an agent is a service** — one runtime
where every endpoint is automatically a tool, an agent registers and is discovered and
load-balanced like anything else, and workflows are durable code paths rather than a
graph DSL. The premise is that the line between "your services" and "your agents" is
accidental complexity; remove it and there's less to wire and keep in sync.

| | Go Micro | tRPC-Agent-Go |
|---|----------|---------------|
| **Primary unit** | A harnessed service (an agent is a service with an LLM inside) | An agent |
| **Orchestration** | Durable `flow` steps + `Loop` — plain code paths | Graph / Chain / Parallel / Cycle agents (graph DSL) |
| **Services as tools** | Every endpoint is automatically an MCP tool | Function tools + MCP, wired explicitly |
| **Service runtime** | Built in — agents *are* services (registry, RPC, load balancing, pub/sub) | Runs alongside your existing service stack (tRPC) |
| **MCP / A2A** | Both, generated from the registry | Both |
| **Evaluation / self-evolution** | Verification loop on the roadmap; not yet first-class | First-class today |
| **Memory / RAG** | Store-backed memory (Postgres, NATS KV, file); RAG on the roadmap | In-memory / Redis memory; RAG today |
| **Observability** | OpenTelemetry run timelines, `micro runs` | OpenTelemetry, Langfuse examples |
| **Backing** | Independent, community | tRPC-Group / Tencent |

### When to choose tRPC-Agent-Go
- You want a graph/workflow DSL for composing agents and tools
- You're on tRPC, or want to add agents alongside an existing service stack
- You want first-class evaluation and self-evolution today, with a large team behind it

### When to choose Go Micro
- You want one runtime where services, agents, and flows are the same primitives —
  registered, discoverable, and deployed the same way
- You want your existing services to become agent tools with zero extra code
- You prefer durable flows and plain code paths over a graph DSL, in a small,
  independent framework you can hold in your head

### They interoperate

Both speak **MCP** and **A2A**, so a Go Micro agent and a tRPC-Agent-Go agent can call
each other over A2A, and either can consume the other's MCP tools. You can run Go Micro
as the service-and-agent runtime and still reach an agent built on tRPC-Agent-Go.

## Feature Deep Dive

### Service Discovery

**Go Micro**: Built-in with plugins
```go
// Zero-config for dev
svc := micro.NewService("myservice")

// Consul for production
reg := consul.NewRegistry()
svc := micro.NewService("myservice", micro.Registry(reg))
```

**go-kit**: Bring your own
```go
// You implement service discovery
// Can be 100+ lines of code
```

**gRPC**: No built-in discovery
```go
// Use external solution like Consul
// or service mesh like Istio
```

### Load Balancing

**Go Micro**: Client-side, pluggable strategies
```go
// Built-in: random, round-robin
selector := selector.NewSelector(
    selector.SetStrategy(selector.RoundRobin),
)
```

**go-kit**: Manual implementation
```go
// You implement load balancing
// Using loadbalancer package
```

**gRPC**: Via external load balancer
```bash
# Use external LB like Envoy, nginx
```

### Pub/Sub

**Go Micro**: First-class
```go
broker.Publish("topic", &broker.Message{Body: []byte("data")})
broker.Subscribe("topic", handler)
```

**go-kit**: Not provided
```go
// Use external message broker directly
// NATS, Kafka, etc
```

**gRPC**: Streaming only
```go
// Use bidirectional streams
// Not traditional pub/sub
```

## Migration Paths

See specific migration guides:
- [From gRPC](migration/from-grpc.md)

**Coming Soon:**
- From go-kit
- From Standard Library

## Decision Matrix

Choose **Go Micro** if:
- ✅ Building Go microservices
- ✅ Want fast iteration
- ✅ Need service discovery
- ✅ Want pub/sub built-in
- ✅ Prefer conventions

Choose **go-kit** if:
- ✅ Want maximum control
- ✅ Have strong architectural opinions
- ✅ Building custom framework
- ✅ Prefer explicit composition

Choose **gRPC** if:
- ✅ Need multi-language support
- ✅ Performance is primary concern
- ✅ Just need RPC (not full framework)
- ✅ Have service discovery handled

Choose **Dapr** if:
- ✅ Polyglot services
- ✅ Heavy Kubernetes usage
- ✅ Want portable cloud abstractions
- ✅ Need state management

## Performance

Rough benchmarks (requests/sec, single instance):

| Framework | Simple RPC | With Discovery | With Tracing |
|-----------|-----------|----------------|--------------|
| Go Micro | ~20k | ~18k | ~15k |
| gRPC | ~25k | N/A | ~20k |
| go-kit | ~22k | N/A | ~18k |
| HTTP std | ~30k | N/A | N/A |

*Benchmarks are approximate and vary by configuration*

## Community & Ecosystem

- **Go Micro**: Active, growing plugins
- **gRPC**: Huge, multi-language
- **go-kit**: Mature, stable
- **Dapr**: Growing, Microsoft-backed

## Recommendation

Start with **Go Micro** if you're building Go microservices and want to move fast. You can always:
- Use gRPC transport: `micro.Transport(grpc.NewTransport())`
- Integrate with go-kit components
- Mix and match as needed

The pluggable architecture means you're not locked in.
