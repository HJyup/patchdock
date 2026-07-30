package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/HJyup/patchdock/internal/tui"
)

func RunPrompt(ctx context.Context) error {
	if !tui.Interactive(os.Stdin, os.Stdout) {
		return errors.New("no task given, and this is not a terminal to ask on — pass one with `dock run -p \"...\"`")
	}

	prompt, err := tui.Prompt(os.Stdin, os.Stdout)
	if errors.Is(err, tui.ErrPromptCancelled) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read task: %w", err)
	}

	return RunTask(ctx, prompt)
}
