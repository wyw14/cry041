package application

import (
	"context"
	"time"

	"github.com/wyw14/cry041/internal/domain"
)

type ReleaseRepository interface {
	Create(context.Context, domain.Release, string) (domain.Release, error)
	Get(context.Context, string) (domain.Release, error)
	Save(context.Context, domain.Release, int64) error
	List(context.Context, ReleaseFilter) ([]domain.Release, int, error)
}

type TemplateRepository interface {
	SaveTemplate(context.Context, domain.TemplateVersion) error
	GetTemplate(context.Context, string, int) (domain.TemplateVersion, error)
}

type AuditRepository interface {
	Append(context.Context, domain.AuditEvent) (domain.AuditEvent, error)
	Head(context.Context, string) (string, error)
	ListAudit(context.Context, string) ([]domain.AuditEvent, error)
}

type ExecutionRepository interface {
	SaveExecution(context.Context, domain.Execution) error
	ListExecutions(context.Context, string) ([]domain.Execution, error)
}

type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }

type AutomationAdapter interface {
	Validate(context.Context, domain.ChecklistItem, string) (string, error)
}

type DeploymentAdapter interface {
	Run(context.Context, domain.Release) (failureCode, failureMessage, recoveryGuide string, err error)
}

type ReleaseFilter struct {
	State          domain.ReleaseState
	OwnerID        string
	Risk           domain.Risk
	From, To       *time.Time
	Page, PageSize int
	Sort           string
}
