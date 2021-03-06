package git

import (
	"context"
	"os"
	"os/exec"
)

func Tag(ctx context.Context, tag, message string) error {
	cmd := exec.CommandContext(ctx, "git", "tag", "-a", tag, "-m", message)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func Add(ctx context.Context, file string) error {
	cmd := exec.CommandContext(ctx, "git", "add", file)
	return cmd.Run()
}

func Commit(ctx context.Context, msg string) error {
	cmd := exec.CommandContext(ctx, "git", "commit", "-m", msg)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
