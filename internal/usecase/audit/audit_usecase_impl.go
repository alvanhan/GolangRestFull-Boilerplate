package audit

import (
	"bytes"
	"context"
	"encoding/csv"
	"time"

	"file-management-service/internal/domain/entity"
	"file-management-service/internal/domain/errors"
	"file-management-service/internal/domain/repository"

	"github.com/google/uuid"
)

type useCaseImpl struct {
	auditRepo repository.AuditRepository
}

func NewUseCase(auditRepo repository.AuditRepository) UseCase {
	return &useCaseImpl{auditRepo: auditRepo}
}

func (uc *useCaseImpl) List(
	ctx context.Context,
	filter *AuditListFilter,
) ([]*AuditLogResponse, int64, error) {
	repoFilter := buildRepoFilter(filter)
	logs, total, err := uc.auditRepo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, errors.InternalServer(err)
	}
	resp := make([]*AuditLogResponse, len(logs))
	for i, l := range logs {
		resp[i] = toAuditLogResponse(l)
	}
	return resp, total, nil
}

func (uc *useCaseImpl) GetByID(ctx context.Context, id string) (*AuditLogResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.BadRequest("invalid audit log ID")
	}
	log, err := uc.auditRepo.GetByID(ctx, uid)
	if err != nil || log == nil {
		return nil, errors.NotFound("audit log")
	}
	return toAuditLogResponse(log), nil
}

func (uc *useCaseImpl) Export(ctx context.Context, filter *AuditListFilter) ([]byte, error) {
	exportFilter := *filter
	exportFilter.Page = 1
	exportFilter.PageSize = 100000

	repoFilter := buildRepoFilter(&exportFilter)
	logs, _, err := uc.auditRepo.List(ctx, repoFilter)
	if err != nil {
		return nil, errors.InternalServer(err)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	headers := []string{"ID", "UserID", "Action", "ResourceID", "ResourceType", "IPAddress", "Status", "CreatedAt"}
	if err := w.Write(headers); err != nil {
		return nil, errors.InternalServer(err)
	}

	for _, l := range logs {
		userID := ""
		if l.UserID != nil {
			userID = l.UserID.String()
		}
		resourceID := ""
		if l.ResourceID != nil {
			resourceID = l.ResourceID.String()
		}
		resourceType := ""
		if l.ResourceType != nil {
			resourceType = *l.ResourceType
		}
		row := []string{
			l.ID.String(),
			userID,
			string(l.Action),
			resourceID,
			resourceType,
			l.IPAddress,
			l.Status,
			l.CreatedAt.Format(time.RFC3339),
		}
		if err := w.Write(row); err != nil {
			return nil, errors.InternalServer(err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, errors.InternalServer(err)
	}
	return buf.Bytes(), nil
}

// buildRepoFilter maps the use-case filter to the repository filter type.
func buildRepoFilter(filter *AuditListFilter) repository.AuditFilter {
	repoFilter := repository.AuditFilter{
		IPAddress: filter.IPAddress,
		Status:    filter.Status,
		Page:      filter.Page,
		PageSize:  filter.PageSize,
	}
	if repoFilter.Page <= 0 {
		repoFilter.Page = 1
	}
	if repoFilter.PageSize <= 0 {
		repoFilter.PageSize = 20
	}

	if filter.UserID != "" {
		if uid, err := uuid.Parse(filter.UserID); err == nil {
			repoFilter.UserID = &uid
		}
	}
	if filter.ResourceID != "" {
		if rid, err := uuid.Parse(filter.ResourceID); err == nil {
			repoFilter.ResourceID = &rid
		}
	}
	if filter.ResourceType != "" {
		rt := filter.ResourceType
		repoFilter.ResourceType = &rt
	}
	if filter.Action != "" {
		action := entity.AuditAction(filter.Action)
		repoFilter.Action = &action
	}
	if filter.StartDate != "" {
		if t, err := time.Parse(time.RFC3339, filter.StartDate); err == nil {
			repoFilter.StartDate = &t
		}
	}
	if filter.EndDate != "" {
		if t, err := time.Parse(time.RFC3339, filter.EndDate); err == nil {
			repoFilter.EndDate = &t
		}
	}
	return repoFilter
}

func toAuditLogResponse(l *entity.AuditLog) *AuditLogResponse {
	resp := &AuditLogResponse{
		ID:           l.ID.String(),
		Action:       string(l.Action),
		IPAddress:    l.IPAddress,
		UserAgent:    l.UserAgent,
		Status:       l.Status,
		ErrorMessage: l.ErrorMessage,
		DurationMs:   l.Duration,
		CreatedAt:    l.CreatedAt,
		ResourceName: l.ResourceName,
	}
	if l.UserID != nil {
		s := l.UserID.String()
		resp.UserID = &s
	}
	if l.ResourceID != nil {
		s := l.ResourceID.String()
		resp.ResourceID = &s
	}
	if l.ResourceType != nil {
		resp.ResourceType = l.ResourceType
	}
	return resp
}
