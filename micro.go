package micro

// Config is used to store configuration about the environment
type Config struct {
	Registry     string
	RegistryAddr string

	Store     string
	StoreAddr string
}

func DefaultConsulConfig() Config {
	return Config{
		Registry: "consul",
		Store:    "consul",
	}
}
