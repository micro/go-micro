package resource

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"go-micro.dev/v5/config"
	"go-micro.dev/v5/config/source/env"
)

// configCommand exposes the config interface: get, dump.
//
// The CLI loads configuration from environment variables (the source
// that makes sense without a running service). Keys use dot notation,
// e.g. "database.host" reads from DATABASE_HOST.
func configCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Read dynamic configuration (from environment)",
		Description: `Read dynamic configuration loaded from environment variables.

Keys use dot notation: "database.host" maps to DATABASE_HOST.

  micro config get <key>    Read a config value
  micro config dump         Print the full config as JSON`,
		Subcommands: []*cli.Command{
			{
				Name:      "get",
				Usage:     "Read a config value",
				ArgsUsage: "<key>",
				Action:    configGet,
			},
			{
				Name:   "dump",
				Usage:  "Print the full config",
				Action: configDump,
			},
		},
	}
}

func loadConfig() (config.Config, error) {
	conf, err := config.NewConfig()
	if err != nil {
		return nil, err
	}
	if err := conf.Load(env.NewSource()); err != nil {
		return nil, err
	}
	return conf, nil
}

func configGet(c *cli.Context) error {
	key := c.Args().First()
	if key == "" {
		return fail("usage: micro config get <key>")
	}
	conf, err := loadConfig()
	if err != nil {
		return fail("load config: %v", err)
	}
	path := strings.Split(key, ".")
	val, err := conf.Get(path...)
	if err != nil {
		return fail("get %q: %v", key, err)
	}
	fmt.Println(string(val.Bytes()))
	return nil
}

func configDump(c *cli.Context) error {
	conf, err := loadConfig()
	if err != nil {
		return fail("load config: %v", err)
	}
	fmt.Println(string(conf.Bytes()))
	return nil
}
