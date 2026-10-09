package chat

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/google/uuid"
	"go-micro.dev/v6/store"
)

func (s *session) interactive(ctx context.Context) error {
	if !readline.IsTerminal(int(os.Stdin.Fd())) {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "exit" || line == "quit" || line == "/exit" {
				return nil
			}
			if line == "" {
				continue
			}
			if line == "reset" || line == "/new" {
				if err := s.selectSession(uuid.NewString()); err != nil {
					return err
				}
				continue
			}
			if err := s.ask(ctx, line); err != nil {
				return err
			}
		}
		return scanner.Err()
	}
	s.interactiveUI = true
	defer func() { s.interactiveUI = false }()
	s.stream = true
	terminal, err := readline.NewEx(&readline.Config{
		Prompt: "micro > ",
		AutoComplete: readline.NewPrefixCompleter(
			readline.PcItem("/new"), readline.PcItem("/sessions"),
			readline.PcItem("/resume"), readline.PcItem("/stop"),
			readline.PcItem("/approve"), readline.PcItem("/deny"), readline.PcItem("/help"), readline.PcItem("/exit")),
	})
	if err != nil {
		return err
	}
	defer terminal.Close()
	s.output = terminal.Stdout()
	defer func() { s.output = nil }()
	fmt.Fprintf(s.writer(), "Session %s\n/new · /sessions · /resume ID · /stop · /exit\n", s.id)
	if err := s.showHistory(); err != nil {
		return err
	}
	var cancel context.CancelFunc
	var done <-chan error
	defer func() {
		if cancel != nil {
			cancel()
			<-done
		}
	}()
	for {
		line, err := terminal.Readline()
		if errors.Is(err, readline.ErrInterrupt) {
			if cancel != nil {
				cancel()
			}
			continue
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if done != nil {
			select {
			case <-done:
				cancel()
				cancel, done = nil, nil
			default:
			}
		}
		if line == "/exit" || line == "exit" || line == "quit" {
			return nil
		}
		if line == "/stop" {
			if cancel != nil {
				cancel()
			}
			continue
		}
		if line == "/approve" || line == "/deny" {
			s.answerApproval(line == "/approve")
			continue
		}
		if line == "/help" {
			fmt.Fprintln(s.writer(), "Use /approve or /deny for a proposed tool action. Enter a request. Ctrl-C or /stop cancels current work. /new starts a conversation; /sessions lists saved conversations; /resume ID reopens one. Exiting stops local work.")
			continue
		}
		if cancel != nil {
			fmt.Fprintln(s.writer(), "A request is running. Use /stop before sending another request.")
			continue
		}
		switch {
		case line == "/new" || line == "reset":
			if err := s.selectSession(uuid.NewString()); err != nil {
				return err
			}
			fmt.Fprintf(s.writer(), "Session %s\n", s.id)
		case strings.HasPrefix(line, "/resume "):
			if err := s.selectSession(strings.TrimSpace(strings.TrimPrefix(line, "/resume "))); err != nil {
				return err
			}
			fmt.Fprintf(s.writer(), "Session %s\n", s.id)
			if err := s.showHistory(); err != nil {
				return err
			}
		case line == "/sessions":
			keys, err := s.conversations().List(store.ListPrefix("session/"))
			if err != nil {
				return err
			}
			for _, key := range keys {
				records, err := s.conversations().Read(key)
				if err != nil {
					return err
				}
				for _, record := range records {
					var entry conversation
					if err := record.Decode(&entry); err != nil {
						return err
					}
					fmt.Fprintf(s.writer(), "%s  %s\n", entry.ID, entry.Title)
				}
			}
		default:
			cancel, done = s.startPrompt(ctx, line)
		}
	}
}

func (s *session) startPrompt(ctx context.Context, line string) (context.CancelFunc, <-chan error) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		err := s.ask(ctx, line)
		if err != nil {
			fmt.Fprintf(s.writer(), "Error: %v\n", err)
		}
		done <- err
	}()
	return cancel, done
}
