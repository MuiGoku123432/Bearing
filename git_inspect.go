package bearing

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Git and hooks can print arbitrary output. Drain it while keeping tool context bounded.
type gitOutput struct {
	buffer    bytes.Buffer
	truncated bool
}

func (out *gitOutput) Write(data []byte) (int, error) {
	n := len(data)
	remaining := 65536 - out.buffer.Len()
	if len(data) > remaining {
		out.truncated = true
		data = data[:remaining]
	}
	_, err := out.buffer.Write(data)
	return n, err
}

func runGit(ctx context.Context, path string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", path}, args...)...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	command.WaitDelay = 2 * time.Second
	var stdout, stderr gitOutput
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if err != nil {
		message := strings.TrimSpace(stderr.buffer.String())
		if len(message) > 2000 {
			message = message[:2000] + " (truncated)"
		}
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, message)
	}
	if stdout.truncated {
		return nil, fail("io_error", "Git output exceeded 64 KiB. Narrow the repository operation before retrying.")
	}
	return stdout.buffer.Bytes(), nil
}

func (s *Store) inspectGit(ctx context.Context, result GitResult) (GitResult, error) {
	branches, err := runGit(ctx, s.Root, "for-each-ref", "--format=%(refname:short)", "--sort=refname", "refs/heads/")
	if err != nil {
		return result, err
	}
	if text := strings.TrimSpace(string(branches)); text != "" {
		result.Branches = strings.Split(text, "\n")
	}
	if len(result.Branches) > 100 {
		result.Branches = result.Branches[:100]
		result.Warnings = append(result.Warnings, "Only the first 100 branches are shown.")
	}
	result.Trunk = trunk(ctx, s.Root)
	index, err := runGit(ctx, s.Root, "write-tree")
	if err != nil {
		result.Warnings = append(result.Warnings, "The staged tree could not be read. Resolve index conflicts before committing.")
	} else {
		result.IndexTree = strings.TrimSpace(string(index))
	}
	staged, err := runGit(ctx, s.Root, "diff", "--cached", "--stat=80,40,20", "--no-ext-diff", "--no-textconv", "--no-renames", "--")
	if err != nil {
		return result, err
	}
	result.StagedSummary = string(staged)
	result.Message = "Start new work on a Conventional Branch from the trunk. Review the staged diff, then commit with index_tree as expected_index so exactly that content is committed."
	return result, nil
}
