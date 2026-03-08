package audit

import (
	"context"
	"time"
)

// AuditLogResponse is the public representation of an audit log entry.
type AuditLogResponse struct {
	ID           string     `json:"id"`
	Action       string     `json:"action"`
	IPAddress    string     `json:"ip_address"`
	UserAgent    string     `json:"user_agent"`
	Status       string     `json:"status"`
	UserID       *string    `json:"user_id,omitempty"`
	ResourceID   *string    `json:"resource_id,omitempty"`
	ResourceType *string    `json:"resource_type,omitempty"`
	ResourceName *string    `json:"resource_name,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	DurationMs   int64      `json:"duration_ms"`
	CreatedAt    time.Time  `json:"created_at"`
}

// AuditListFilter carries query parameters for listing audit logs.
type AuditListFilter struct {
	UserID       string `query:"user_id"`
	Action       string `query:"action"`
	ResourceID   string `query:"resource_id"`
	ResourceType string `query:"resource_type"`
	IPAddress    string `query:"ip_address"`
	Status       string `query:"status"`
	StartDate    string `query:"start_date"`
	EndDate      string `query:"end_date"`
	Page         int    `query:"page"`
	PageSize     int    `query:"page_size"`
}

// UseCase is the audit business-logic contract.
type UseCase interface {
	// List returns a paginated list of audit log entries matching the filter.
	List(ctx context.Context, filter *AuditListFilter) ([]*AuditLogResponse, int64, error)

	// GetByID returns a single audit log entry by ID.
	GetByID(ctx context.Context, id string) (*AuditLogResponse, error)

	// Export returns a CSV-encoded byte slice of matching audit log entries.
	Export(ctx context.Context, filter *AuditListFilter) ([]byte, error)
}
