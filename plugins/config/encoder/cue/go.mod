module github.com/asim/go-micro/plugins/config/encoder/cue/v4

go 1.17

require (
	cuelang.org/go v0.0.15
	github.com/ghodss/yaml v1.0.0
	github.com/stretchr/testify v1.7.0
	go-micro.dev/v4 v4.2.1
)

require (
	github.com/cockroachdb/apd/v2 v2.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mpvl/unique v0.0.0-20150818121801-cbe035fff7de // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.0.0-20210510120150-4163338589ed // indirect
	golang.org/x/text v0.3.6 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace go-micro.dev/v4 => ../../../../../go-micro
