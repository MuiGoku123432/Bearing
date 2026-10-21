package bearing

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

func validateCommitMessage(message string) error {
	if len(message) == 0 || len(message) > 8192 || !utf8.ValidString(message) || strings.ContainsAny(message, "\x00\r") {
		return fail("invalid_input", "Use a UTF-8 commit message of 1–8192 bytes with LF line endings and no NUL.")
	}
	lines := strings.Split(message, "\n")
	match := regexp.MustCompile(`^[a-z][a-z0-9-]*(\([^()\r\n]+\))?!?: \S.*$`).FindStringSubmatch(lines[0])
	if match == nil || (match[1] != "" && strings.TrimSpace(strings.Trim(match[1], "()")) == "") {
		return fail("invalid_input", "Use type(scope): description, type: description, or type(scope)!: description. Use a lowercase type such as feat, fix, docs, test, refactor, or chore.")
	}
	if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
		return fail("invalid_input", "Separate the commit subject from its body or footers with a blank line.")
	}
	for _, line := range lines[1:] {
		for _, prefix := range []string{"BREAKING CHANGE:", "BREAKING-CHANGE:"} {
			if strings.HasPrefix(line, prefix) && strings.TrimSpace(strings.TrimPrefix(line, prefix)) == "" {
				return fail("invalid_input", "Describe the breaking change after its footer label.")
			}
		}
	}
	return nil
}

func (s *Store) commitGit(ctx context.Context, in GitInput, result GitResult) (GitResult, error) {
	if err := validateCommitMessage(in.Message); err != nil {
		return result, err
	}
	before := result.Repository
	if before.Branch == "" {
		return result, fail("conflict", "Create or switch to a branch before committing; this checkout has detached HEAD.")
	}
	if before.Branch == "main" || before.Branch == "master" || before.Branch == "develop" || before.Branch == trunk(ctx, s.Root) {
		return result, fail("conflict", "Bearing does not commit directly to the trunk branch "+before.Branch+". Create a branch first; staged and uncommitted changes move with it.")
	}
	index, err := runGit(ctx, s.Root, "write-tree")
	if err != nil {
		return result, fail("conflict", "Resolve staged conflicts before committing.")
	}
	result.IndexTree = strings.TrimSpace(string(index))
	if in.ExpectedIndex == "" || in.ExpectedIndex != result.IndexTree {
		return result, fail("conflict", "The staged changes differ from expected_index. Inspect and review the staged diff again.")
	}
	currentTree, err := runGit(ctx, s.Root, "rev-parse", "HEAD^{tree}")
	if err != nil {
		return result, err
	}
	if strings.TrimSpace(string(currentTree)) == result.IndexTree {
		return result, fail("invalid_input", "There are no staged changes to commit.")
	}
	result.Message = "Commit only the staged changes with message: " + strings.SplitN(in.Message, "\n", 2)[0]
	if in.DryRun {
		return result, nil
	}
	messagePath := filepath.Join(s.Root, ".bearing", "commit-"+NewID()+".txt")
	file, err := os.OpenFile(messagePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return result, err
	}
	defer os.Remove(messagePath)
	_, writeErr := file.WriteString(in.Message)
	closeErr := file.Close()
	if writeErr != nil {
		return result, writeErr
	}
	if closeErr != nil {
		return result, closeErr
	}
	// note: repeat the staged-tree check just before Git runs its normal hooks.
	latest, err := runGit(ctx, s.Root, "write-tree")
	if err != nil || strings.TrimSpace(string(latest)) != in.ExpectedIndex {
		return result, fail("conflict", "The staged tree changed while preparing the commit. Inspect again.")
	}
	if _, err := runGit(ctx, s.Root, "commit", "--cleanup=verbatim", "--file", messagePath); err != nil {
		return result, fail("outcome_unknown", "Git did not confirm the commit. Inspect HEAD and staged changes before retrying; hooks may have run: "+err.Error())
	}
	current, err := ResolveRepository(ctx, s.Root)
	if err != nil {
		return result, fail("outcome_unknown", "Git reported a successful commit but the final HEAD could not be read. Inspect before retrying.")
	}
	result.Repository = current
	result.Message = "Created commit " + current.Head
	actualMessage, messageErr := runGit(ctx, s.Root, "log", "-1", "--format=%B")
	if messageErr != nil || validateCommitMessage(string(actualMessage)) != nil {
		result.Warnings = append(result.Warnings, "The final message could not be verified as conventional. A hook may have rewritten it; inspect the existing commit before taking further action.")
	}
	actualTree, treeErr := runGit(ctx, s.Root, "rev-parse", "HEAD^{tree}")
	if treeErr != nil || strings.TrimSpace(string(actualTree)) != in.ExpectedIndex || current.Branch != before.Branch || current.Head == before.Head {
		result.Warnings = append(result.Warnings, "The resulting commit differs from the reviewed staged tree or branch. A hook or another Git process may have changed it; inspect before proceeding.")
	}
	return result, nil
}
