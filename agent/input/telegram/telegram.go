package telegram

import (
	"errors"
	"strings"
	"sync"

	"github.com/micro/cli/v2"
	"github.com/micro/go-micro/v2/agent/input"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type telegramInput struct {
	sync.Mutex

	debug     bool
	token     string
	whitelist []string

	api *tgbotapi.BotAPI
}

type ChatType string

const (
	Private    ChatType = "private"
	Group      ChatType = "group"
	Supergroup ChatType = "supergroup"
)

func init() {
	input.Inputs["telegram"] = &telegramInput{}
}

func (ti *telegramInput) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "telegram_debug",
			EnvVars: []string{"MICRO_TELEGRAM_DEBUG"},
			Usage:   "Telegram debug output",
		},
		&cli.StringFlag{
			Name:    "telegram_token",
			EnvVars: []string{"MICRO_TELEGRAM_TOKEN"},
			Usage:   "Telegram token",
		},
		&cli.StringFlag{
			Name:    "telegram_whitelist",
			EnvVars: []string{"MICRO_TELEGRAM_WHITELIST"},
			Usage:   "Telegram bot's users (comma-separated values)",
		},
	}
}

func (ti *telegramInput) Init(ctx *cli.Context) error {
	ti.debug = ctx.Bool("telegram_debug")
	ti.token = ctx.String("telegram_token")

	whitelist := ctx.String("telegram_whitelist")

	if whitelist != "" {
		ti.whitelist = strings.Split(whitelist, ",")
	}

	if len(ti.token) == 0 {
		return errors.New("missing telegram token")
	}

	return nil
}

func (ti *telegramInput) Stream() (input.Conn, error) {
	ti.Lock()
	defer ti.Unlock()

	return newConn(ti)
}

func (ti *telegramInput) Start() error {
	ti.Lock()
	defer ti.Unlock()

	api, err := tgbotapi.NewBotAPI(ti.token)
	if err != nil {
		return err
	}

	ti.api = api

	api.Debug = ti.debug

	return nil
}

func (ti *telegramInput) Stop() error {
	return nil
}

func (p *telegramInput) String() string {
	return "telegram"
}
