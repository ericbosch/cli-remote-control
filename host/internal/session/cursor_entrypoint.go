package session

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ericbosch/cli-remote-control/host/internal/policy"
)

type cursorEngineEntrypoint struct {
	Name                        string
	Bin                         string
	ArgsPrefix                  []string
	SupportsStructuredStreaming bool
	SupportsPromptFlag          bool
}

func detectCursorEngineEntrypoint(ctx context.Context) (cursorEngineEntrypoint, error) {
	// Precedence:
	//  1) cursor-agent
	//  2) agent
	//  3) cursor agent (only if Cursor IDE truly installed and subcommand help works)
	if _, err := exec.LookPath("cursor-agent"); err == nil {
		ep := cursorEngineEntrypoint{Name: "cursor-agent", Bin: "cursor-agent"}
		ep.SupportsStructuredStreaming, ep.SupportsPromptFlag = cursorHelpCaps(ctx, ep.Bin, nil)
		return ep, nil
	}
	if _, err := exec.LookPath("agent"); err == nil {
		ep := cursorEngineEntrypoint{Name: "agent", Bin: "agent"}
		ep.SupportsStructuredStreaming, ep.SupportsPromptFlag = cursorHelpCaps(ctx, ep.Bin, nil)
		return ep, nil
	}
	if _, err := exec.LookPath("cursor"); err == nil {
		if cursorIDEInstalled(ctx) && cursorAgentSubcommandWorks(ctx) {
			ep := cursorEngineEntrypoint{Name: "cursor agent", Bin: "cursor", ArgsPrefix: []string{"agent"}}
			ep.SupportsStructuredStreaming, ep.SupportsPromptFlag = cursorHelpCaps(ctx, ep.Bin, ep.ArgsPrefix)
			return ep, nil
		}
	}
	return cursorEngineEntrypoint{}, errors.New("cursor engine entrypoint not found (need cursor-agent, agent, or Cursor IDE)")
}

func cursorIDEInstalled(ctx context.Context) bool {
	c, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	out, err := runCombined(c, "cursor", "--version")
	if err != nil {
		return false
	}
	low := strings.ToLower(string(out))
	if strings.Contains(low, "no cursor ide installation found") ||
		(strings.Contains(low, "cursor ide") && strings.Contains(low, "not found")) ||
		strings.Contains(low, "no installation found") {
		return false
	}
	return true
}

func cursorAgentSubcommandWorks(ctx context.Context) bool {
	c, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err := runCombined(c, "cursor", "agent", "--help")
	return err == nil
}

func cursorHelpCaps(ctx context.Context, bin string, prefix []string) (structured bool, prompt bool) {
	c, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	args := append([]string{}, prefix...)
	args = append(args, "--help")
	out, err := runCombined(c, bin, args...)
	if err != nil {
		return false, false
	}
	s := strings.ToLower(string(out))
	structured = strings.Contains(s, "output-format") && strings.Contains(s, "stream-json")
	prompt = strings.Contains(s, " -p") || strings.Contains(s, "--prompt")
	return structured, prompt
}

func runCombined(ctx context.Context, bin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	env, _ := policy.EngineEnv(os.Environ())
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	// Normalize context timeout errors to a plain error.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out, context.DeadlineExceeded
	}
	// Some CLIs write their normal output to stderr; keep combined output for string checks.
	return bytes.TrimSpace(out), err
}
