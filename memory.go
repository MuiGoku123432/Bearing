package bearing

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
)

type Anchor struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type Memory struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	Claim      string     `json:"claim"`
	Basis      string     `json:"basis"`
	Source     string     `json:"source"`
	Scope      string     `json:"scope"`
	Anchors    []Anchor   `json:"anchors,omitempty"`
	Status     string     `json:"status"`
	Version    int        `json:"version"`
	Supersedes string     `json:"supersedes,omitempty"`
	Reason     string     `json:"reason,omitempty"`
	CreatedAt  string     `json:"created_at"`
	UpdatedAt  string     `json:"updated_at"`
	Context    Repository `json:"context"`
}

type RememberInput struct {
	RequestID       string   `json:"request_id"`
	Action          string   `json:"action,omitempty"`
	ID              string   `json:"id,omitempty"`
	ExpectedVersion int      `json:"expected_version,omitempty"`
	Kind            string   `json:"kind,omitempty"`
	Claim           string   `json:"claim,omitempty"`
	Basis           string   `json:"basis,omitempty"`
	Source          string   `json:"source,omitempty"`
	Scope           string   `json:"scope,omitempty"`
	Anchors         []Anchor `json:"anchors,omitempty"`
	Reason          string   `json:"reason,omitempty"`
}

func (s *Store) Remember(ctx context.Context, in RememberInput) (Mutation, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	var repo Repository
	return s.write(ctx, in.RequestID, "remember", in, func() error {
		if in.Action == "" {
			in.Action = "save"
		}
		if in.Action != "save" && in.Action != "supersede" && in.Action != "retract" {
			return fail("invalid_input", "Memory action must be save, supersede, or retract.")
		}
		if in.Action != "save" && (in.ID == "" || in.ExpectedVersion < 1 || strings.TrimSpace(in.Reason) == "") {
			return fail("invalid_input", "Replacing or retracting memory needs its ID, version, and a reason.")
		}
		if in.Action == "save" && (in.ID != "" || in.ExpectedVersion != 0) {
			return fail("invalid_input", "New memories must not supply an existing ID or version.")
		}
		if in.Action != "retract" {
			if strings.TrimSpace(in.Claim) == "" || strings.TrimSpace(in.Source) == "" {
				return fail("invalid_input", "Memory needs a claim and its source.")
			}
			if in.Basis != "user" && in.Basis != "observed" && in.Basis != "inferred" {
				return fail("invalid_input", "Memory basis must be user, observed, or inferred.")
			}
			if in.Kind != "decision" && in.Kind != "constraint" && in.Kind != "lesson" && in.Kind != "preference" {
				return fail("invalid_input", "Memory kind must be decision, constraint, lesson, or preference.")
			}
			if in.Scope == "" {
				in.Scope = "project"
			}
			if in.Scope != "project" && in.Scope != "checkout" {
				return fail("invalid_input", "Memory scope must be project or checkout.")
			}
			if len(in.Anchors) > 16 {
				return fail("invalid_input", "Use at most sixteen memory anchors.")
			}
			for _, a := range in.Anchors {
				if (a.Kind != "path" && a.Kind != "symbol") || strings.TrimSpace(a.Value) == "" {
					return fail("invalid_input", "Anchors need a path or symbol kind and a value.")
				}
			}
		}
		if err := inputSize(in, 4096); err != nil {
			return err
		}
		var err error
		repo, err = ResolveRepository(ctx, s.Root)
		return err
	}, func(tx *sql.Tx) (Mutation, error) {
		if in.Action != "save" {
			var data string
			var old Memory
			if err := tx.QueryRowContext(ctx, "SELECT data FROM memories WHERE id=?", in.ID).Scan(&data); err == sql.ErrNoRows {
				return Mutation{}, fail("not_found", "Memory not found.")
			} else if err != nil {
				return Mutation{}, err
			}
			if err := json.Unmarshal([]byte(data), &old); err != nil {
				return Mutation{}, err
			}
			if old.Version != in.ExpectedVersion {
				return Mutation{}, &Error{Code: "conflict", Message: "Memory changed; reload before retrying.", CurrentVersion: old.Version}
			}
			if old.Status != "active" {
				return Mutation{}, fail("conflict", "Only active memory can be replaced or retracted.")
			}
			old.Version++
			old.Status, old.Reason, old.UpdatedAt = "superseded", in.Reason, repo.CapturedAt
			if in.Action == "retract" {
				old.Status = "retracted"
			}
			encoded, err := json.Marshal(old)
			if err != nil {
				return Mutation{}, err
			}
			if _, err := tx.ExecContext(ctx, "UPDATE memories SET status=?,version=?,data=? WHERE id=?", old.Status, old.Version, string(encoded), old.ID); err != nil {
				return Mutation{}, err
			}
			if _, err := tx.ExecContext(ctx, "DELETE FROM memory_search WHERE id=?", old.ID); err != nil {
				return Mutation{}, err
			}
			if in.Action == "retract" {
				return Mutation{ID: old.ID, Version: old.Version, Status: old.Status}, nil
			}
		}
		memory := Memory{ID: NewID(), Kind: in.Kind, Claim: in.Claim, Basis: in.Basis, Source: in.Source, Scope: in.Scope, Anchors: in.Anchors, Status: "active", Version: 1, Reason: in.Reason, CreatedAt: repo.CapturedAt, UpdatedAt: repo.CapturedAt, Context: repo}
		if in.Action == "supersede" {
			memory.Supersedes = in.ID
		}
		data, err := json.Marshal(memory)
		if err != nil {
			return Mutation{}, err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO memories(id,status,version,data) VALUES(?,?,?,?)", memory.ID, memory.Status, memory.Version, string(data)); err != nil {
			return Mutation{}, err
		}
		for _, a := range memory.Anchors {
			if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO memory_anchors(memory_id,kind,value) VALUES(?,?,?)", memory.ID, a.Kind, a.Value); err != nil {
				return Mutation{}, err
			}
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO memory_search(id,claim,evidence) VALUES(?,?,?)", memory.ID, memory.Claim, memory.Source); err != nil {
			return Mutation{}, err
		}
		return Mutation{ID: memory.ID, Version: 1, Status: memory.Status}, nil
	})
}
