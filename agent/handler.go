package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pb "go-micro.dev/v6/agent/proto"
	"go-micro.dev/v6/server"
	"go-micro.dev/v6/service"
)

// Register attaches this agent to a service's server and lifecycle.
// Use service.Handle(agent.New(...)); the service owns infrastructure and shutdown.
func (a *agentImpl) Register(o *service.Options) error {
	a.lifecycle.Lock()
	defer a.lifecycle.Unlock()
	if a.attached || a.server != nil {
		return errors.New("agent: already hosted")
	}
	if a.opts.A2AAddress != "" {
		return errors.New("agent: direct A2A hosting requires standalone Run")
	}
	a.attached = true
	var cancel context.CancelFunc
	o.BeforeStart = append(o.BeforeStart, func() error {
		a.lifecycle.Lock()
		defer a.lifecycle.Unlock()
		a.tasks.mu.Lock()
		stopped := a.tasks.stopped
		a.tasks.mu.Unlock()
		if stopped {
			return errors.New("agent: host has stopped")
		}
		a.opts.Name = o.Server.Options().Name
		a.opts.Registry = o.Server.Options().Registry
		a.opts.Client, a.opts.Store, a.opts.Broker = o.Client, o.Store, o.Broker
		if a.opts.Services == nil {
			a.opts.Services = []string{a.opts.Name}
		}
		a.tools, a.mem = nil, nil
		a.setup()
		if a.model == nil {
			return fmt.Errorf("agent: unknown model provider %q", a.opts.Provider)
		}
		a.server = o.Server
		if err := a.registerHandlers(a.server); err != nil {
			return err
		}
		o.Context, cancel = context.WithCancel(o.Context)
		return nil
	})
	o.AfterStart = append(o.AfterStart, func() error {
		a.lifecycle.Lock()
		defer a.lifecycle.Unlock()
		a.startHost()
		go func() {
			<-a.schedules.done
			cancel()
		}()
		return nil
	})
	o.BeforeStop = append(o.BeforeStop, a.Stop)
	o.AfterStop = append(o.AfterStop, func() error {
		if cancel != nil {
			cancel()
		}
		if err := a.schedules.err; err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("agent schedules: %w", err)
		}
		return nil
	})
	return nil
}

func (a *agentImpl) registerHandlers(srv server.Server) error {
	if srv.Options().Metadata["agent"] != "" {
		return errors.New("agent: service already hosts an agent")
	}
	sessionSupport := "v1"
	if a.opts.Memory != nil {
		if _, ok := a.opts.Memory.(SessionMemory); !ok {
			sessionSupport = ""
		}
	}
	modelName := a.opts.Model
	if a.model != nil {
		modelName = a.model.Options().Model
	}
	md := make(map[string]string)
	for k, v := range srv.Options().Metadata {
		md[k] = v
	}
	md["agent"] = "v1"
	if md["type"] == "" {
		md["type"] = "agent"
	}
	md["interaction"], md["sessions"], md["session_history"] = "v1", sessionSupport, sessionSupport
	md["tasks"], md["schedules"] = sessionSupport, sessionSupport
	md["provider"], md["model"], md["services"] = a.opts.Provider, modelName, strings.Join(a.opts.Services, ",")
	if err := srv.Init(server.Metadata(md)); err != nil {
		return err
	}
	if err := pb.RegisterAgentHandler(srv, a); err != nil {
		return err
	}
	if err := pb.RegisterAgentInteractionHandler(srv, &interactionHandler{agent: a}); err != nil {
		return err
	}
	if err := pb.RegisterAgentSessionsHandler(srv, &sessionHandler{agent: a}); err != nil {
		return err
	}

	if err := pb.RegisterAgentTasksHandler(srv, &taskHandler{agent: a}); err != nil {
		return err
	}

	if err := pb.RegisterAgentSchedulesHandler(srv, &scheduleHandler{agent: a}); err != nil {
		return err
	}
	return nil
}

// startHost starts work owned by the host after the RPC server is listening.
// The caller holds lifecycle.
func (a *agentImpl) startHost() <-chan struct{} {
	a.tasks.mu.Lock()
	a.tasks.address = a.server.Options().Address
	a.tasks.mu.Unlock()
	stopCh := make(chan struct{})
	a.stopCh = stopCh
	ctx, cancel := context.WithCancel(context.Background())
	a.schedules.cancel = cancel
	a.schedules.done = make(chan struct{})
	go func() {
		a.schedules.err = a.runSchedules(ctx)
		close(a.schedules.done)
	}()
	return stopCh
}

// Agent control endpoints must never be offered back to a model as tools.
func agentEndpoint(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) < 3 {
		return false
	}
	switch parts[len(parts)-2] {
	case "Agent", "AgentInteraction", "AgentSessions", "AgentTasks", "AgentSchedules":
		return true
	}
	return false
}
