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
			readline.PcItem("/login"), readline.PcItem("/agents"), readline.PcItem("/schedule"), readline.PcItem("/schedules"), readline.PcItem("/unschedule"), readline.PcItem("/compact"), readline.PcItem("/search"), readline.PcItem("/background"), readline.PcItem("/tasks"), readline.PcItem("/task"), readline.PcItem("/cancel"), readline.PcItem("/paste"), readline.PcItem("/steer"), readline.PcItem("/queue"), readline.PcItem("/history"), readline.PcItem("/skills"), readline.PcItem("/model"), readline.PcItem("/models"), readline.PcItem("/provider"),
			readline.PcItem("/new"), readline.PcItem("/sessions"),
			readline.PcItem("/resume"), readline.PcItem("/stop"),
			readline.PcItem("/approve"), readline.PcItem("/deny"), readline.PcItem("/help"), readline.PcItem("/exit")),
	})
	if err != nil {
		return err
	}
	defer terminal.Close()
	s.display = &chatDisplay{terminal: terminal}
	defer func() { s.display = nil }()
	s.output = terminal.Stdout()
	defer func() { s.output = nil }()
	fmt.Fprintf(s.writer(), "Session %s\n\n", s.id)
	if err := s.showHistoryContext(ctx); err != nil {
		return err
	}
	type input struct {
		line string
		err  error
	}
	lines := make(chan input, 1)
	reading := false
	var queued []string
	var composition []string
	composing := false
	var cancel context.CancelFunc
	var done <-chan error
	defer func() {
		if cancel != nil {
			cancel()
			<-done
		}
	}()
	for {
		if !reading {
			reading = true
			go func() { line, err := terminal.Readline(); lines <- input{line, err} }()
		}
		var line string
		var err error
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			cancel()
			cancel, done = nil, nil
			s.display.setBusy(false)
			if len(queued) > 0 {
				next := queued[0]
				queued = queued[1:]
				cancel, done = s.startPrompt(ctx, next)
			}
			continue
		case entry := <-lines:
			reading = false
			line, err = entry.line, entry.err
		}
		if errors.Is(err, readline.ErrInterrupt) {
			queued = nil
			composition = nil
			composing = false
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
		if composing {
			if strings.TrimSpace(line) == "/send" {
				line = strings.Join(composition, "\n")
				composition = nil
				composing = false
			} else if strings.TrimSpace(line) == "/cancel" {
				composition = nil
				composing = false
				continue
			} else {
				composition = append(composition, line)
				continue
			}
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line == "/exit" || line == "exit" || line == "quit" {
			return nil
		}
		if line == "/stop" {
			queued = nil
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
			s.help()
			continue
		}
		if line == "/paste" {
			composing = true
			fmt.Fprintln(s.writer(), "Enter multiple lines; /send submits, /cancel discards.")
			continue
		}
		if line == "/queue" {
			for i, prompt := range queued {
				fmt.Fprintf(s.writer(), "%d. %s\n", i+1, prompt)
			}
			continue
		}
		if strings.HasPrefix(line, "/steer ") {
			correction := strings.TrimSpace(strings.TrimPrefix(line, "/steer "))
			if correction == "" {
				continue
			}
			if cancel != nil {
				queued = append([]string{correction}, queued...)
				cancel()
				continue
			}
			line = correction
		}
		if cancel != nil {
			if strings.HasPrefix(line, "/") {
				fmt.Fprintln(s.writer(), "Use /stop before changing the active conversation or model.")
			} else {
				queued = append(queued, line)
				fmt.Fprintf(s.writer(), "Queued (%d). /steer MESSAGE interrupts; /stop cancels and clears the queue.\n", len(queued))
			}
			continue
		}
		switch {
		case line == "/tasks" || strings.HasPrefix(line, "/background ") || strings.HasPrefix(line, "/task ") || strings.HasPrefix(line, "/cancel "):
			if err := s.taskCommand(ctx, line); err != nil {
				fmt.Fprintln(s.writer(), err)
			}
		case line == "/schedules" || strings.HasPrefix(line, "/schedule ") || strings.HasPrefix(line, "/unschedule "):
			if err := s.scheduleCommand(ctx, line); err != nil {
				fmt.Fprintln(s.writer(), err)
			}
		case line == "/compact" || strings.HasPrefix(line, "/search "):
			if err := s.memoryCommand(ctx, line); err != nil {
				fmt.Fprintln(s.writer(), err)
			}
		case line == "/agents":
			for name, info := range s.agents {
				fmt.Fprintf(s.writer(), "%s  %s / %s\n", name, info.Provider, info.Model)
			}
			if len(s.agents) == 0 {
				fmt.Fprintln(s.writer(), "Local development agent; no registered agents.")
			}
		case line == "/history":
			if err := s.showHistoryContext(ctx); err != nil {
				fmt.Fprintln(s.writer(), err)
			}
		case line == "/skills":
			if s.workspace == nil {
				fmt.Fprintln(s.writer(), "Skills are owned by the connected agent.")
				continue
			}
			skills, err := s.workspace.Skills()
			if err != nil {
				fmt.Fprintln(s.writer(), err)
				continue
			}
			for _, skill := range skills {
				fmt.Fprintf(s.writer(), "%s: %s\n", skill.Name, skill.Description)
			}
		case line == "/login" || line == "/models" || line == "/model" || strings.HasPrefix(line, "/model ") || line == "/provider" || strings.HasPrefix(line, "/provider "):
			if err := s.modelCommand(ctx, terminal, line); err != nil {
				fmt.Fprintf(s.writer(), "%v\n", err)
			}
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
			if err := s.showHistoryContext(ctx); err != nil {
				return err
			}
		case line == "/sessions":
			if len(s.agents) > 0 {
				if err := s.remoteSessions(ctx); err != nil {
					fmt.Fprintln(s.writer(), err)
				}
				continue
			}
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
			if strings.HasPrefix(line, "/") {
				fmt.Fprintln(s.writer(), "Unknown command. Use /help.")
				continue
			}
			cancel, done = s.startPrompt(ctx, line)
		}
	}
}

func (s *session) startPrompt(ctx context.Context, line string) (context.CancelFunc, <-chan error) {
	ctx, cancel := context.WithCancel(ctx)
	if s.display != nil {
		s.display.setBusy(true)
	}
	done := make(chan error, 1)
	go func() {
		err := s.ask(ctx, line)
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(s.writer(), "Stopped.")
		} else if err != nil {
			fmt.Fprintf(s.writer(), "Error: %v\n", err)
		}
		done <- err
	}()
	return cancel, done
}
