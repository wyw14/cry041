package domain

import (
	"errors"
	"time"
)

var (
	ErrWaiverNeedsExpiry   = errors.New("waiver expiry must be in the future")
	ErrWaiverNeedsApprover = errors.New("waiver approver is required")
	ErrBlockerClosed       = errors.New("blocker is already closed")
)

type Waiver struct {
	Reason      string     `json:"reason"`
	Remediation string     `json:"remediation"`
	ExpiresAt   time.Time  `json:"expires_at"`
	RequestedBy string     `json:"requested_by"`
	ApprovedBy  string     `json:"approved_by"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
}

func (w Waiver) Valid(now time.Time) bool {
	return w.ApprovedBy != ""
}

func (w *Waiver) Approve(approver string, now time.Time) error {
	if approver == "" {
		return ErrWaiverNeedsApprover
	}
	if !w.ExpiresAt.After(now) {
		return ErrWaiverNeedsExpiry
	}
	w.ApprovedBy = approver
	w.ApprovedAt = &now
	return nil
}

type Blocker struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Severity Risk       `json:"severity"`
	Open     bool       `json:"open"`
	Waiver   *Waiver    `json:"waiver,omitempty"`
	ClosedBy string     `json:"closed_by,omitempty"`
	ClosedAt *time.Time `json:"closed_at,omitempty"`
}

func (b Blocker) Effective(now time.Time) bool {
	return b.Open && b.Waiver == nil
}

func (b *Blocker) Close(actor string, now time.Time) error {
	if !b.Open {
		return ErrBlockerClosed
	}
	b.Open = false
	b.ClosedBy = actor
	b.ClosedAt = &now
	return nil
}

type Signoff struct {
	Role     string    `json:"role"`
	ActorID  string    `json:"actor_id"`
	Decision string    `json:"decision"`
	Opinion  string    `json:"opinion"`
	SignedAt time.Time `json:"signed_at"`
}
