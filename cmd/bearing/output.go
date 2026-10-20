package main

import (
	"encoding/json"
	"fmt"
	"io"

	bearing "github.com/MuiGoku123432/Bearing"
)

func printHuman(w io.Writer, value any) error {
	switch out := value.(type) {
	case string:
		_, err := fmt.Fprintln(w, out)
		return err
	case bearing.Policy:
		_, err := fmt.Fprintf(w, "%s\n\n%s\n\nPolicy: %s\nWorkflow: %s\n", out.Text, out.Workflow, out.ID, out.WorkflowID)
		return err
	case bearing.Mutation:
		_, err := fmt.Fprintf(w, "%s: %s (version %d)\nRequest: %s\n", out.Status, out.ID, out.Version, out.RequestID)
		if out.Replayed {
			fmt.Fprintln(w, "This request was already saved.")
		}
		for _, id := range out.DocumentIDs {
			fmt.Fprintln(w, "Saved document:", id)
		}
		return err
	case bearing.StartResult:
		if err := printHuman(w, out.Mutation); err != nil {
			return err
		}
		return printHuman(w, out.Context)
	case bearing.Context:
		fmt.Fprintln(w, out.Policy.Text)
		if out.Task != nil {
			fmt.Fprintf(w, "\nTask: %s (%s, %s, version %d)\n%s\n", out.Task.ID, out.Task.Kind, out.Task.Status, out.Task.Version, out.Task.Outcome)
			for _, criterion := range out.Task.Acceptance {
				fmt.Fprintf(w, "- %s: %s\n", criterion.ID, criterion.Description)
			}
		}
		if out.Checkpoint != nil {
			fmt.Fprintln(w, "\nSaved progress:", out.Checkpoint.Summary)
		}
		for _, doc := range out.Documents {
			fmt.Fprintf(w, "Document: %s [%s] %s\n", doc.Path, doc.ID, doc.Summary)
		}
		for _, memory := range out.Memories {
			fmt.Fprintf(w, "Memory: %s [%s; %s]\n", memory.Claim, memory.ID, memory.Basis)
		}
		for _, candidate := range out.Candidates {
			fmt.Fprintf(w, "%s: %s\n", candidate.ID, candidate.Outcome)
		}
		for _, warning := range out.Warnings {
			fmt.Fprintln(w, "Note:", warning)
		}
		_, err := fmt.Fprintln(w, "Next:", out.Next)
		return err
	case []bearing.Task:
		if len(out) == 0 {
			_, err := fmt.Fprintln(w, "No active tasks.")
			return err
		}
		for _, task := range out {
			if _, err := fmt.Fprintf(w, "%s  %s  v%d  %s\n", task.ID, task.Status, task.Version, task.Outcome); err != nil {
				return err
			}
		}
		return nil
	case bearing.RecallResult:
		for _, memory := range out.Memories {
			fmt.Fprintf(w, "%s (v%d, %s, %s)\n%s\nSource: %s\n\n", memory.ID, memory.Version, memory.Status, memory.Basis, memory.Claim, memory.Source)
		}
		for _, doc := range out.Documents {
			fmt.Fprintf(w, "%s [%s]\n%s\n", doc.Path, doc.ID, doc.Body)
		}
		for _, warning := range out.Warnings {
			fmt.Fprintln(w, "Note:", warning)
		}
		if len(out.Memories) == 0 && len(out.Documents) == 0 {
			_, err := fmt.Fprintln(w, "No matching records.")
			return err
		}
		return nil
	default:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	}
}
