package generate

type Config struct {
	Service  string
	Vendor   string
	Dir      string
	Comments []string
	Client   bool
	Jaeger   bool
	Skaffold bool
}
