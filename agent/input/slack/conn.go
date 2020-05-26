package slack

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/micro/go-micro/v2/agent/input"
	"github.com/nlopes/slack"
)

// Satisfies the input.Conn interface
type slackConn struct {
	auth *slack.AuthTestResponse
	rtm  *slack.RTM
	exit chan bool

	sync.Mutex
	names map[string]string
}

func (s *slackConn) run() {
	// func retrieves user names and maps to IDs
	setNames := func() {
		names := make(map[string]string)
		users, err := s.rtm.Client.GetUsers()
		if err != nil {
			return
		}

		for _, user := range users {
			names[user.ID] = user.Name
		}

		s.Lock()
		s.names = names
		s.Unlock()
	}

	setNames()

	t := time.NewTicker(time.Minute)
	defer t.Stop()

	for {
		select {
		case <-s.exit:
			return
		case <-t.C:
			setNames()
		}
	}
}

func (s *slackConn) getName(id string) string {
	s.Lock()
	name := s.names[id]
	s.Unlock()
	return name
}

func (s *slackConn) Close() error {
	select {
	case <-s.exit:
		return nil
	default:
		close(s.exit)
	}
	return nil
}

func (s *slackConn) Recv(event *input.Event) error {
	if event == nil {
		return errors.New("event cannot be nil")
	}

	for {
		select {
		case <-s.exit:
			return errors.New("connection closed")
		case e := <-s.rtm.IncomingEvents:
			switch ev := e.Data.(type) {
			case *slack.MessageEvent:
				// only accept type message
				if ev.Type != "message" {
					continue
				}

				// only accept DMs or messages to me
				switch {
				case strings.HasPrefix(ev.Channel, "D"):
				case strings.HasPrefix(ev.Text, s.auth.User):
				case strings.HasPrefix(ev.Text, fmt.Sprintf("<@%s>", s.auth.UserID)):
				default:
					continue
				}

				// Strip username from text
				switch {
				case strings.HasPrefix(ev.Text, s.auth.User):
					args := strings.Split(ev.Text, " ")[1:]
					ev.Text = strings.Join(args, " ")
					event.To = s.auth.User
				case strings.HasPrefix(ev.Text, fmt.Sprintf("<@%s>", s.auth.UserID)):
					args := strings.Split(ev.Text, " ")[1:]
					ev.Text = strings.Join(args, " ")
					event.To = s.auth.UserID
				}

				if event.Meta == nil {
					event.Meta = make(map[string]interface{})
				}

				// fill in the blanks
				event.From = ev.Channel + ":" + ev.User
				event.Type = input.TextEvent
				event.Data = []byte(ev.Text)
				event.Meta["reply"] = ev
				return nil
			case *slack.InvalidAuthEvent:
				return errors.New("invalid credentials")
			}
		}
	}
}

func (s *slackConn) Send(event *input.Event) error {
	var channel, message, name string

	if len(event.To) == 0 {
		return errors.New("require Event.To")
	}

	parts := strings.Split(event.To, ":")

	if len(parts) == 2 {
		channel = parts[0]
		name = s.getName(parts[1])
		// try using reply meta
	} else if ev, ok := event.Meta["reply"]; ok {
		channel = ev.(*slack.MessageEvent).Channel
		name = s.getName(ev.(*slack.MessageEvent).User)
	}

	// don't know where to send the message
	if len(channel) == 0 {
		return errors.New("could not determine who message is to")
	}

	if len(name) == 0 || strings.HasPrefix(channel, "D") {
		message = string(event.Data)
	} else {
		message = fmt.Sprintf("@%s: %s", name, string(event.Data))
	}

	s.rtm.SendMessage(s.rtm.NewOutgoingMessage(message, channel))
	return nil
}
