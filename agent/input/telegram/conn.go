package telegram

import (
	"errors"
	"strings"
	"sync"

	"github.com/forestgiant/sliceutil"
	"github.com/micro/go-micro/v2/agent/input"
	"github.com/micro/go-micro/v2/logger"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

type telegramConn struct {
	input *telegramInput

	recv <-chan tgbotapi.Update
	exit chan bool

	syncCond *sync.Cond
	mutex    sync.Mutex
}

func newConn(input *telegramInput) (*telegramConn, error) {
	conn := &telegramConn{
		input: input,
	}

	conn.syncCond = sync.NewCond(&conn.mutex)

	go conn.run()

	return conn, nil
}

func (tc *telegramConn) run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := tc.input.api.GetUpdatesChan(u)
	if err != nil {
		return
	}

	tc.recv = updates
	tc.syncCond.Signal()

	select {
	case <-tc.exit:
		return
	}
}

func (tc *telegramConn) Close() error {
	return nil
}

func (tc *telegramConn) Recv(event *input.Event) error {
	if event == nil {
		return errors.New("event cannot be nil")
	}

	for {
		if tc.recv == nil {
			tc.mutex.Lock()
			tc.syncCond.Wait()
		}

		update := <-tc.recv

		if update.Message == nil || (len(tc.input.whitelist) > 0 && !sliceutil.Contains(tc.input.whitelist, update.Message.From.UserName)) {
			continue
		}

		if event.Meta == nil {
			event.Meta = make(map[string]interface{})
		}

		event.Type = input.TextEvent
		event.From = update.Message.From.UserName
		event.To = tc.input.api.Self.UserName
		event.Data = []byte(update.Message.Text)
		event.Meta["chatId"] = update.Message.Chat.ID
		event.Meta["chatType"] = update.Message.Chat.Type
		event.Meta["messageId"] = update.Message.MessageID

		return nil
	}
}

func (tc *telegramConn) Send(event *input.Event) error {
	messageText := strings.TrimSpace(string(event.Data))

	chatId := event.Meta["chatId"].(int64)
	chatType := ChatType(event.Meta["chatType"].(string))

	msgConfig := tgbotapi.NewMessage(chatId, messageText)
	msgConfig.ParseMode = tgbotapi.ModeHTML

	if sliceutil.Contains([]ChatType{Group, Supergroup}, chatType) {
		msgConfig.ReplyToMessageID = event.Meta["messageId"].(int)
	}

	_, err := tc.input.api.Send(msgConfig)

	if err != nil {
		// probably it could be because of nested HTML tags -- telegram doesn't allow nested tags
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Error("[telegram][Send] error:", err)
		}
		msgConfig.Text = "This bot couldn't send the response (Internal error)"
		tc.input.api.Send(msgConfig)
	}

	return nil
}
