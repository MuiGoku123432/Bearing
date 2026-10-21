package bearing

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// write provides one receipt/transaction contract for all five mutations.
func (s *Store) write(ctx context.Context, id, operation string, input any, prepare func() error, apply func(*sql.Tx) (Mutation, error)) (Mutation, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	if id == "" {
		// The receipt returns the generated ID, so an uncertain result can still be retried.
		id = NewID()
	}
	if strings.TrimSpace(id) == "" || len(id) > 128 {
		return Mutation{}, fail("invalid_input", "Use a request_id of at most 128 bytes, or omit it.")
	}
	if err := inputSize(input, 65536); err != nil {
		return Mutation{}, err
	}
	data, err := json.Marshal(input)
	if err != nil {
		return Mutation{}, err
	}
	// The request ID is the receipt key; the digest covers only the requested change,
	// so retrying with a generated ID matches the original omitted-ID request.
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(data, &fields); err != nil {
		return Mutation{}, err
	}
	delete(fields, "request_id")
	if data, err = json.Marshal(fields); err != nil {
		return Mutation{}, err
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	// The first receipt check avoids repeating file capture after a lost response.
	var savedOp, savedDigest, saved string
	err = s.DB.QueryRowContext(ctx, "SELECT operation,digest,outcome FROM write_receipts WHERE request_id=?", id).Scan(&savedOp, &savedDigest, &saved)
	if err == nil {
		return receipt(id, operation, digest, savedOp, savedDigest, saved)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Mutation{}, err
	}
	if prepare != nil {
		if err = prepare(); err != nil {
			return Mutation{}, err
		}
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Mutation{}, err
	}
	defer tx.Rollback()
	// note: the repeated query closes the interprocess race after preparation.
	err = tx.QueryRowContext(ctx, "SELECT operation,digest,outcome FROM write_receipts WHERE request_id=?", id).Scan(&savedOp, &savedDigest, &saved)
	if err == nil {
		return receipt(id, operation, digest, savedOp, savedDigest, saved)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Mutation{}, err
	}
	result, err := apply(tx)
	if err != nil {
		return Mutation{}, err
	}
	result.RequestID = id
	result.PolicyID = Rules().ID
	data, err = json.Marshal(result)
	if err != nil {
		return Mutation{}, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO write_receipts(request_id,operation,digest,outcome) VALUES(?,?,?,?)", id, operation, digest, string(data)); err != nil {
		return Mutation{}, err
	}
	if err = tx.Commit(); err != nil {
		return Mutation{}, &Error{Code: "outcome_unknown", Message: "The write result could not be confirmed. Retry with the same request ID.", Retryable: true, RequestID: id}
	}
	return result, nil
}

func receipt(id, op, digest, savedOp, savedDigest, saved string) (Mutation, error) {
	if op != savedOp || digest != savedDigest {
		return Mutation{}, fail("conflict", "This request_id was already used for a different write. Use a new request_id for a new action, or omit it.")
	}
	var result Mutation
	if err := json.Unmarshal([]byte(saved), &result); err != nil {
		return Mutation{}, err
	}
	result.Replayed = true
	result.RequestID = id
	result.PolicyID = Rules().ID
	return result, nil
}

func taskInTransaction(ctx context.Context, tx *sql.Tx, id string, version int) (Task, error) {
	var data string
	if err := tx.QueryRowContext(ctx, "SELECT data FROM tasks WHERE id=?", id).Scan(&data); errors.Is(err, sql.ErrNoRows) {
		return Task{}, fail("not_found", "Task not found in this checkout.")
	} else if err != nil {
		return Task{}, err
	}
	var task Task
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return Task{}, err
	}
	if task.Version != version {
		return Task{}, &Error{Code: "conflict", Message: "This task changed in another client. Reload it before saving.", CurrentVersion: task.Version}
	}
	if task.Status == "completed" || task.Status == "cancelled" {
		return Task{}, fail("conflict", "This task is closed. Start a follow-up task instead.")
	}
	return task, nil
}
