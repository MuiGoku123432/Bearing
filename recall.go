package bearing

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"unicode"
)

type RecallInput struct {
	Query       string   `json:"query,omitempty"`
	IDs         []string `json:"ids,omitempty"`
	DocumentIDs []string `json:"document_ids,omitempty"`
	Anchors     []Anchor `json:"anchors,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	BudgetBytes int      `json:"budget_bytes,omitempty"`
}

type RecallResult struct {
	Memories  []Memory   `json:"memories"`
	Documents []Document `json:"documents,omitempty"`
	Omitted   int        `json:"omitted"`
	Warnings  []string   `json:"warnings,omitempty"`
	PolicyID  string     `json:"policy_id"`
}

func (s *Store) Recall(ctx context.Context, in RecallInput) (RecallResult, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	out := RecallResult{Memories: []Memory{}, PolicyID: Rules().ID}
	if err := inputSize(in, 16000); err != nil {
		return out, err
	}
	if in.Limit == 0 {
		in.Limit = 5
	}
	if in.BudgetBytes == 0 {
		in.BudgetBytes = 12000
	}
	if in.Limit < 1 || in.Limit > 20 || in.BudgetBytes < 1024 || in.BudgetBytes > 300000 || len(in.IDs) > 20 || len(in.DocumentIDs) > 8 || len(in.Anchors) > 16 {
		return out, fail("invalid_input", "Recall accepts up to twenty memories, eight document IDs, and a 1–300 kB budget.")
	}
	if len(in.DocumentIDs) > 0 {
		if len(in.IDs) > 0 || in.Query != "" {
			return out, fail("invalid_input", "Retrieve documents separately from memories.")
		}
		for _, id := range in.DocumentIDs {
			doc, err := s.document(ctx, id)
			if err != nil {
				return out, err
			}
			data, err := json.Marshal(doc)
			if err != nil {
				return out, err
			}
			if len(data) > in.BudgetBytes {
				doc.Body = ""
				out.Omitted++
				out.Warnings = append(out.Warnings, "Document body omitted by the byte budget: "+id)
			} else {
				in.BudgetBytes -= len(data)
			}
			out.Documents = append(out.Documents, doc)
			if warning := s.documentWarning(doc); warning != "" {
				out.Warnings = append(out.Warnings, warning)
			}
		}
		return out, nil
	}
	query := "SELECT m.data FROM memories m WHERE m.status='active' AND (json_extract(m.data,'$.scope')='project' OR json_extract(m.data,'$.context.path')=?)"
	args := []any{s.Root}
	rowLimit := in.Limit + 1
	if len(in.IDs) > 0 {
		args = nil
		seen := map[string]bool{}
		for _, id := range in.IDs {
			if !seen[id] {
				seen[id] = true
				args = append(args, id)
			}
		}
		// Read every requested row so a missing ID is detected even when the limit omits results.
		rowLimit = len(args)
		query = "SELECT m.data FROM memories m WHERE m.id IN (" + strings.TrimSuffix(strings.Repeat("?,", rowLimit), ",") + ")"
	} else {
		if len(in.Anchors) > 0 {
			query += " AND (NOT EXISTS(SELECT 1 FROM memory_anchors a WHERE a.memory_id=m.id) OR EXISTS(SELECT 1 FROM memory_anchors a WHERE a.memory_id=m.id AND ("
			var clauses []string
			for _, a := range in.Anchors {
				clauses = append(clauses, "(a.kind=? AND a.value=?)")
				args = append(args, a.Kind, a.Value)
			}
			query += strings.Join(clauses, " OR ") + ")))"
		}
		if in.Query != "" {
			words := strings.FieldsFunc(in.Query, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
			if len(words) == 0 {
				// note: same hint as the zero-result warning below; this path never queries.
				out.Warnings = append(out.Warnings, "No memories matched. Add path or symbol anchors, or recall explicit memory IDs, if you expect a saved record.")
				return out, nil
			}
			if len(words) > 30 {
				words = words[:30]
			}
			for i := range words {
				words[i] = "\"" + words[i] + "\""
			}
			query = strings.Replace(query, "FROM memories m WHERE", "FROM memories m JOIN memory_search ON memory_search.id=m.id WHERE", 1)
			query += " AND memory_search MATCH ?"
			args = append(args, strings.Join(words, " OR "))
		}
	}
	query += " ORDER BY "
	if in.Query != "" && len(in.IDs) == 0 {
		query += "bm25(memory_search),"
	}
	query += "json_extract(m.data,'$.updated_at') DESC,m.id LIMIT ?"
	args = append(args, rowLimit)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var data string
		var memory Memory
		if err := rows.Scan(&data); err != nil {
			return out, err
		}
		if err := json.Unmarshal([]byte(data), &memory); err != nil {
			return out, err
		}
		if len(out.Memories) >= in.Limit || len(data) > in.BudgetBytes {
			out.Omitted++
			continue
		}
		in.BudgetBytes -= len(data)
		out.Memories = append(out.Memories, memory)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if out.Omitted > 0 {
		out.Warnings = append(out.Warnings, "More results exist or exceeded the budget; narrow the query or retrieve IDs explicitly.")
	}
	if len(out.Memories) == 0 && out.Omitted == 0 {
		out.Warnings = append(out.Warnings, "No memories matched. Add path or symbol anchors, or recall explicit memory IDs, if you expect a saved record.")
	}
	if len(in.IDs) > 0 && len(out.Memories)+out.Omitted < rowLimit {
		return out, fail("not_found", "One or more memory IDs were not found.")
	}
	return out, nil
}

// loadJSON is shared by task, checkpoint, and history reads.
func (s *Store) loadJSON(ctx context.Context, query string, id string, target any) error {
	var data string
	err := s.DB.QueryRowContext(ctx, query, id).Scan(&data)
	if err == sql.ErrNoRows {
		return fail("not_found", "Record not found.")
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), target)
}
