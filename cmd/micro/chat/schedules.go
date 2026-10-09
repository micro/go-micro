package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-micro.dev/v6/agent"
	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/client"
)

func (s *session) scheduleCommand(ctx context.Context, line string) error {
	if len(s.agents) != 1 {
		return fmt.Errorf("schedules need one running agent host; use micro chat --host NAME, then connect to NAME")
	}
	for name, info := range s.agents {
		if !info.Schedules {
			return fmt.Errorf("%s does not advertise scheduling", name)
		}
		api := pb.NewAgentSchedulesService(name, s.cl)
		switch {
		case line == "/schedules":
			response, err := api.List(ctx, &pb.ListSchedulesRequest{})
			if err != nil {
				return err
			}
			for _, entry := range response.Schedules {
				fmt.Fprintf(s.writer(), "%s  every %s  next %s  task=%s\n  %s\n", entry.Id, entry.Every, time.Unix(entry.Next, 0).Format(time.RFC3339), entry.LastTask, entry.Message)
				if entry.Error != "" {
					fmt.Fprintln(s.writer(), entry.Error)
				}
			}
		case strings.HasPrefix(line, "/unschedule "):
			_, err := api.Remove(ctx, &pb.RemoveScheduleRequest{Id: strings.TrimSpace(strings.TrimPrefix(line, "/unschedule "))})
			if err != nil {
				return err
			}
			fmt.Fprintln(s.writer(), "Schedule removed. An already-running task is unaffected.")
		default:
			every, message, found := strings.Cut(strings.TrimSpace(strings.TrimPrefix(line, "/schedule ")), " ")
			if !found {
				return fmt.Errorf("use /schedule 1h MESSAGE")
			}
			entry, err := api.Add(agent.WithSession(ctx, s.id), &pb.AddScheduleRequest{Message: message, Every: every}, client.WithRetries(0))
			if err != nil {
				return err
			}
			fmt.Fprintf(s.writer(), "Schedule %s saved on %s, every %s. The host must remain running.\n", entry.Id, name, entry.Every)
		}
	}
	return nil
}
