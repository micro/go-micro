package workspace

import (
	"context"
	"os/exec"
	"time"

	"github.com/google/uuid"
)

// WithContainer runs commands through Docker in an isolated container. Only the
// workspace is mounted writable; networking and privileged capabilities are
// disabled. The image must provide sh. Docker must be installed on the host.
func WithContainer(image string) Option { return func(w *Workspace) { w.container = image } }

func (w *Workspace) command(ctx context.Context, command string) *exec.Cmd {
	if w.container == "" {
		return shellCommand(ctx, command)
	}
	name := "micro-" + uuid.NewString()
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--name", name,
		"--network=none", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges",
		"--pids-limit=128", "--memory=512m", "--cpus=1", "--tmpfs", "/tmp:rw,nosuid,size=64m",
		"--mount", "type=bind,src="+w.dir+",dst=/workspace", "--workdir", "/workspace", "--", w.container, "sh", "-lc", command)
	cmd.Cancel = func() error {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = exec.CommandContext(cleanup, "docker", "rm", "--force", name).Run()
		if cmd.Process != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
	return cmd
}
