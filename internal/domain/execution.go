package domain

import "time"

type ExecutionStatus string

const (
	ExecutionRunning   ExecutionStatus = "running"
	ExecutionSucceeded ExecutionStatus = "succeeded"
	ExecutionFailed    ExecutionStatus = "failed"
)

type Execution struct {
	ID             string          `json:"id"`
	ReleaseID      string          `json:"release_id"`
	Status         ExecutionStatus `json:"status"`
	FailureCode    string          `json:"failure_code,omitempty"`
	FailureMessage string          `json:"failure_message,omitempty"`
	RecoveryGuide  string          `json:"recovery_guide,omitempty"`
	StartedAt      time.Time       `json:"started_at"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
}

type AuditEvent struct {
	ID           string    `json:"id"`
	ReleaseID    string    `json:"release_id"`
	ActorID      string    `json:"actor_id"`
	Action       string    `json:"action"`
	Detail       string    `json:"detail"`
	PreviousHash string    `json:"previous_hash"`
	Hash         string    `json:"hash"`
	CreatedAt    time.Time `json:"created_at"`
}
