package discord

import (
	"fmt"
	"sync"

	"errors"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/micro/cli/v2"
	"github.com/micro/go-micro/v2/agent/input"
)

func init() {
	input.Inputs["discord"] = newInput()
}

func newInput() *discordInput {
	return &discordInput{}
}

type discordInput struct {
	token     string
	whitelist []string
	prefix    string
	prefixfn  func(string) (string, bool)
	botID     string

	session *discordgo.Session

	sync.Mutex
	running bool
	exit    chan struct{}
}

func (d *discordInput) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "discord_token",
			EnvVars: []string{"MICRO_DISCORD_TOKEN"},
			Usage:   "Discord token (prefix with Bot if it's for bot account)",
		},
		&cli.StringFlag{
			Name:    "discord_whitelist",
			EnvVars: []string{"MICRO_DISCORD_WHITELIST"},
			Usage:   "Discord Whitelist (seperated by ,)",
		},
		&cli.StringFlag{
			Name:    "discord_prefix",
			Usage:   "Discord Prefix",
			EnvVars: []string{"MICRO_DISCORD_PREFIX"},
			Value:   "Micro ",
		},
	}
}

func (d *discordInput) Init(ctx *cli.Context) error {
	token := ctx.String("discord_token")
	whitelist := ctx.String("discord_whitelist")
	prefix := ctx.String("discord_prefix")

	if len(token) == 0 {
		return errors.New("require token")
	}

	d.token = token
	d.prefix = prefix

	if len(whitelist) > 0 {
		d.whitelist = strings.Split(whitelist, ",")
	}

	return nil
}

func (d *discordInput) Start() error {
	if len(d.token) == 0 {
		return errors.New("missing discord configuration")
	}

	d.Lock()
	defer d.Unlock()

	if d.running {
		return nil
	}

	var err error
	d.session, err = discordgo.New("Bot " + d.token)
	if err != nil {
		return err
	}

	u, err := d.session.User("@me")
	if err != nil {
		return err
	}

	d.botID = u.ID
	d.prefixfn = CheckPrefixFactory(fmt.Sprintf("<@%s> ", d.botID), fmt.Sprintf("<@!%s> ", d.botID), d.prefix)

	d.exit = make(chan struct{})
	d.running = true

	return nil
}

func (d *discordInput) Stream() (input.Conn, error) {
	d.Lock()
	defer d.Unlock()
	if !d.running {
		return nil, errors.New("not running")
	}

	//Fire-n-forget close just in case...
	d.session.Close()

	conn := newConn(d)
	if err := d.session.Open(); err != nil {
		return nil, err
	}
	return conn, nil
}

func (d *discordInput) Stop() error {
	d.Lock()
	defer d.Unlock()

	if !d.running {
		return nil
	}

	close(d.exit)
	d.running = false
	return nil
}

func (d *discordInput) String() string {
	return "discord"
}

// CheckPrefixFactory Creates a prefix checking function and stuff.
func CheckPrefixFactory(prefixes ...string) func(string) (string, bool) {
	return func(content string) (string, bool) {
		for _, prefix := range prefixes {
			if strings.HasPrefix(content, prefix) {
				return strings.TrimPrefix(content, prefix), true
			}
		}
		return "", false
	}
}
