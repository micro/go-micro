package generate

type Config struct {
	Alias    string
	Comments []string
	Dir      string
	Vendor   string
	Client   bool
	Jaeger   bool
	Skaffold bool
}
