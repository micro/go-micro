package chat

import (
	"fmt"
	"sort"
)

func (s *session) welcome() {
	fmt.Fprintln(s.writer(), "\nmicro chat")
	fmt.Fprintf(s.writer(), "Project: %s\n", terminalText(s.project))
	if len(s.agents) == 0 {
		fmt.Fprintf(s.writer(), "Model:   %s / %s\n", s.provider, s.modelName)
		fmt.Fprintln(s.writer(), "Work with project files, commands, skills, and service tools.")
		if s.yes {
			fmt.Fprintln(s.writer(), "Tool actions are allowed without asking (--yes).")
		} else {
			fmt.Fprintln(s.writer(), "I'll ask before changing files, running commands, or making network requests.")
		}
	} else {
		names := make([]string, 0, len(s.agents))
		for name := range s.agents {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(s.writer(), "Agent:   %s\n", terminalText(name))
		}
		fmt.Fprintln(s.writer(), "Connected agents own their tools and approval policy.")
	}
	fmt.Fprintln(s.writer(), "\nDescribe what you need. /help lists controls; Ctrl-C stops work; Ctrl-D exits.")
	if len(s.agents) == 0 {
		fmt.Fprintln(s.writer(), "/provider changes provider; /model changes model; /login replaces the API key.")
	}
}

func (s *session) help() {
	fmt.Fprintln(s.writer(), `Conversation
  /new                  Start a new conversation
  /sessions             List saved conversations
  /resume ID            Resume a conversation
  /history              Show retained messages
  /search TEXT          Search this conversation
  /search --all TEXT    Search recent local conversations
  /compact              Reduce active context

Work
  /approve, /deny        Answer a tool approval request
  /stop or Ctrl-C       Stop work and clear queued requests
  /steer MESSAGE        Stop and give a correction
  /queue                Show queued requests
  /paste                Compose multiple lines; /send submits, /cancel discards
  /skills               List available skills
  /agents               Show connected agents

Model (local agent)
  /provider [NAME [URL]] Select a provider
  /model [ID]           Select a model; /models lists available models
  /login                Replace the current provider's API key

Background work (requires a connected agent host)
  /background MESSAGE   Submit work to the host
  /tasks, /task ID      List tasks or inspect a result
  /cancel ID            Cancel a task
  /schedule 1h MESSAGE  Schedule recurring work
  /schedules            List schedules; /unschedule ID removes one

Requests entered while busy are queued. Exiting stops foreground work.
Completed tool actions are not rolled back. /exit or Ctrl-D exits.`)
}
