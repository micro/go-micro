package slack

import (
	"errors"
	"sync"

	"github.com/micro/cli/v2"
	"github.com/micro/go-micro/v2/agent/input"
	"github.com/nlopes/slack"
)

type slackInput struct {
	debug bool
	token string

	sync.Mutex
	running bool
	exit    chan bool

	api *slack.Client
}

func init() {
	input.Inputs["slack"] = NewInput()
}

func (p *slackInput) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "slack_debug",
			Usage:   "Slack debug output",
			EnvVars: []string{"MICRO_SLACK_DEBUG"},
		},
		&cli.StringFlag{
			Name:    "slack_token",
			Usage:   "Slack token",
			EnvVars: []string{"MICRO_SLACK_TOKEN"},
		},
	}
}

func (p *slackInput) Init(ctx *cli.Context) error {
	debug := ctx.Bool("slack_debug")
	token := ctx.String("slack_token")

	if len(token) == 0 {
		return errors.New("missing slack token")
	}

	p.debug = debug
	p.token = token

	return nil
}

func (p *slackInput) Stream() (input.Conn, error) {
	p.Lock()
	defer p.Unlock()

	if !p.running {
		return nil, errors.New("not running")
	}

	// test auth
	auth, err := p.api.AuthTest()
	if err != nil {
		return nil, err
	}

	rtm := p.api.NewRTM()
	exit := make(chan bool)

	go rtm.ManageConnection()

	go func() {
		select {
		case <-p.exit:
			select {
			case <-exit:
				return
			default:
				close(exit)
			}
		case <-exit:
		}

		rtm.Disconnect()
	}()

	conn := &slackConn{
		auth:  auth,
		rtm:   rtm,
		exit:  exit,
		names: make(map[string]string),
	}

	go conn.run()

	return conn, nil
}

func (p *slackInput) Start() error {
	if len(p.token) == 0 {
		return errors.New("missing slack token")
	}

	p.Lock()
	defer p.Unlock()

	if p.running {
		return nil
	}

	api := slack.New(p.token, slack.OptionDebug(p.debug))

	// test auth
	_, err := api.AuthTest()
	if err != nil {
		return err
	}

	p.api = api
	p.exit = make(chan bool)
	p.running = true
	return nil
}

func (p *slackInput) Stop() error {
	p.Lock()
	defer p.Unlock()

	if !p.running {
		return nil
	}

	close(p.exit)
	p.running = false
	return nil
}

func (p *slackInput) String() string {
	return "slack"
}

func NewInput() input.Input {
	return &slackInput{}
}
