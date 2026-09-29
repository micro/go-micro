---
title: "Apps"
description: "Define and serve an interface over service capabilities."
---

An app gives people an interface to capabilities that agents can also call through
services. The `app` package pairs a portable definition with public assets. A host
supplies service access, authentication, storage and any model-driven generation.

```go
application, err := app.New(app.Definition{
    Name: "notes", Version: "1.0.0",
    Entrypoint: "index.html",
    Services: []app.Dependency{
        {Name: "notes", Endpoints: []string{"Notes.List", "Notes.Add"}},
    },
}, publicAssets)
if err != nil { return err }
mux.Handle("/", application.Handler())
```

`publicAssets` is an `fs.FS` containing only public files. `New` validates the
identity and entrypoint; `CheckDependencies(registry)` checks declared service
endpoints against registered instances. This check does not establish service
health, compatible request schemas or caller permissions.

Use the handler with `net/http` or the existing `web.Service` lifecycle. The host
adds explicit API routes to the same services available as agent tools. The
package does not create a general RPC proxy or grant access from a declaration.

Run the [service-backed app example](https://github.com/micro/go-micro/tree/master/examples/app)
for a complete UI, RPC service and provider-free agent integration test. See the
[package guide](https://github.com/micro/go-micro/tree/master/app) for mounting,
asset behaviour and hosting requirements.

This is an initial v6 app definition, not a generation or deployment system.
Version identifies a release; revision storage remains with the host. Generated
or untrusted apps require host-provided execution isolation and access policy;
static asset serving alone does not provide either.
