package config

import "time"

const (
	Name    = "go.micro.dashboard"
	Version = "1.2.0"
)

type Config struct {
	Server ServerConfig
}

type ServerConfig struct {
	Address string
	Auth    AuthConfig
	CORS    CORSConfig
}

type AuthConfig struct {
	Username        string
	Password        string
	TokenSecret     string
	TokenExpiration time.Duration
}

type CORSConfig struct {
	Enable bool   `toml:"enable"`
	Origin string `toml:"origin"`
}

func GetConfig() Config {
	return *_cfg
}

func GetServerConfig() ServerConfig {
	return _cfg.Server
}

func GetAuthConfig() AuthConfig {
	return _cfg.Server.Auth
}
