# pkger

pkger plugin for `go-config`

### Prerequisites

> Install `pkger` cli

```bash
go install github.com/markbates/pkger/cmd/pkger
pkger -h
```

### Packager

> generating `pkged.go` with all files in `/config` as part of build pipeline

```bash
pkger -o srv/greeter -include /config
```

### Usage

```go
	if err := config.Load(
		pkger.NewSource(pkger.WithPath("/config/config.yaml")),
	); err != nil {
    log.Fatal(err.Error())
	}
```
