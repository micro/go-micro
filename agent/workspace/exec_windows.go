package workspace

import (
	"context"
	"fmt"
	"os/exec"
)

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd", "/C", command)
	cmd.Cancel = func() error { return exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprint(cmd.Process.Pid)).Run() }
	return cmd
}
