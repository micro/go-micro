package chat

import (
	"context"
	"fmt"
	"strings"

	"go-micro.dev/v6/agent"
	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/client"
	"go-micro.dev/v6/store"
)

type taskLocation struct {
	Agent   string
	Address string
}

func (s *session) taskCommand(ctx context.Context, line string) error {
	if strings.HasPrefix(line, "/task ") || strings.HasPrefix(line, "/cancel ") {
		command, id, _ := strings.Cut(line, " ")
		id = strings.TrimSpace(id)
		records, err := s.conversations().Read("task/" + id)
		if err != nil {
			return fmt.Errorf("task location: %w; use /tasks on its host first", err)
		}
		if len(records) == 0 {
			return fmt.Errorf("unknown task %s", id)
		}
		var location taskLocation
		if err := records[0].Decode(&location); err != nil {
			return err
		}
		api := pb.NewAgentTasksService(location.Agent, s.cl)
		var task *pb.TaskResponse
		options := []client.CallOption{client.WithAddress(location.Address)}
		if command == "/cancel" {
			task, err = api.Stop(ctx, &pb.TaskRequest{Id: id}, options...)
		} else {
			task, err = api.Get(ctx, &pb.TaskRequest{Id: id}, options...)
			if err != nil {
				task, err = api.Get(ctx, &pb.TaskRequest{Id: id})
			}
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(s.writer(), "%s  %s\n%s\n", task.Id, task.Status, task.Reply)
		if task.Error != "" {
			fmt.Fprintln(s.writer(), task.Error)
		}
		return nil
	}
	if len(s.agents) != 1 {
		return fmt.Errorf("background work needs one running host: start `micro chat --host NAME` in another terminal, then connect with `micro chat NAME`")
	}
	for name, info := range s.agents {
		if !info.Tasks {
			return fmt.Errorf("%s does not advertise background work", name)
		}
		api := pb.NewAgentTasksService(name, s.cl)
		if line == "/tasks" {
			response, err := api.List(ctx, &pb.ListTasksRequest{})
			if err != nil {
				return err
			}
			for _, task := range response.Tasks {
				if err := s.saveTaskLocation(name, task); err != nil {
					return err
				}
				fmt.Fprintf(s.writer(), "%s  %s  session=%s\n", task.Id, task.Status, task.Session)
			}
			return nil
		}
		message := strings.TrimSpace(strings.TrimPrefix(line, "/background "))
		task, err := api.Start(agent.WithSession(ctx, s.id), &pb.StartTaskRequest{Message: message}, client.WithRetries(0))
		if err != nil {
			return fmt.Errorf("submit task: %w; check /tasks before resubmitting", err)
		}
		// Always show the accepted ID, even if the local locator cannot be saved.
		fmt.Fprintf(s.writer(), "Task %s running on %s. /task %s checks progress; /cancel %s stops it. You can exit chat.\n", task.Id, name, task.Id, task.Id)
		if err := s.saveTaskLocation(name, task); err != nil {
			return err
		}
		return s.remember(message)
	}
	return nil
}

func (s *session) saveTaskLocation(name string, task *pb.TaskResponse) error {
	if task.Address == "" {
		return fmt.Errorf("host omitted its task address")
	}
	return s.conversations().Write(store.NewRecord("task/"+task.Id, taskLocation{Agent: name, Address: task.Address}))
}
