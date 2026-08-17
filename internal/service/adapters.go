package service

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/wyw14/cry041/internal/domain"
)

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }

type SequenceIDs struct{ n atomic.Uint64 }

func (s *SequenceIDs) NewID() string { return fmt.Sprintf("local-%08d", s.n.Add(1)) }

type LocalAutomation struct{}

func (LocalAutomation) Validate(ctx context.Context, item domain.ChecklistItem, value string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if item.Required && value == "" {
		return "failed", nil
	}
	if item.EvidenceRequired && value == "confirmed-without-evidence" {
		return "failed", nil
	}
	return "passed", nil
}

type SimulatedDeployment struct {
	FailCode string
	Delay    time.Duration
}

func (s SimulatedDeployment) Run(ctx context.Context, r domain.Release) (string, string, string, error) {
	if r.State != domain.StateReleased {
		return "NOT_RELEASED", "release is not in released state", "finish all gates before retrying", errors.New("deployment rejected")
	}
	if s.Delay > 0 {
		timer := time.NewTimer(s.Delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return "CANCELLED", ctx.Err().Error(), "check cancellation source and start a new execution", ctx.Err()
		case <-timer.C:
		}
	}
	if s.FailCode != "" {
		return s.FailCode, "local deployment simulation failed", "restore the previous snapshot and verify health checks", errors.New("simulated deployment failure")
	}
	return "", "", "", nil
}
