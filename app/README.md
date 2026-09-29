# App

An app is a human-facing interface to capabilities that agents can also use through services.
This initial v6 package defines an app and serves its public assets. It works independently of Mu.

```go
public, err := fs.Sub(embeddedFiles, "public")
if err != nil { return err }

application, err := app.New(app.Definition{
    Name: "notes", Version: "1.0.0", Description: "Shared notes",
    Entrypoint: "index.html",
    Services: []app.Dependency{
        {Name: "notes", Endpoints: []string{"Notes.List", "Notes.Add"}},
    },
}, public)
if err != nil { return err }

mux.Handle("/", application.Handler())
```

`embeddedFiles` can be an `embed.FS`; any `fs.FS` is supported. The definition is
JSON-serializable so a host can store or advertise it. Treat the filesystem as
immutable for an app version, and supply only assets meant to be public.
`Definition()` returns a copy; changing it does not change the running app.

`Definition.Validate()` checks the shape without a running service. `New` also
checks that the entrypoint exists. `CheckDependencies(reg)` checks that a
registered version of each dependency has all declared endpoints and a node.
It is a point-in-time metadata check, not a health, schema or permission check.

## Serving

Use the handler with `net/http`, or use the existing `web` package for registration
and lifecycle management:

```go
def := application.Definition()
server := web.NewService(
    web.Name(def.Name), web.Version(def.Version),
    web.Address("127.0.0.1:8080"),
    web.Handler(application.Handler()),
)
return server.Run()
```

The entrypoint is served at `/`. Relative asset URLs work when mounted beneath a
prefix with `http.StripPrefix`. Files support GET and HEAD; missing paths and
directories return 404. There is no directory listing or automatic SPA fallback.
The host supplies API routes, authentication and middleware; declaring a service
does not expose it or authorize a caller.

This handler serves trusted app assets. It does not isolate JavaScript or prevent
an OS-backed filesystem from following symlinks. Hosts serving generated or
untrusted apps must supply their own origin/iframe isolation and access policy.

## Scope

The package owns identity, version, description, entrypoint, assets and service
dependency declarations. It does not own generation, persistence, app revisions,
accounts, deployment, or an unrestricted HTTP-to-RPC proxy. Version is an opaque
identifier, not a revision database. Model-generated and hand-written apps use
the same definition.

Run the [example](../examples/app/) to use one service through a UI and an agent.
