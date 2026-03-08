package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"file-management-service/internal/domain/entity"
	domrepo "file-management-service/internal/domain/repository"
	"file-management-service/pkg/pagination"
	"file-management-service/pkg/response"
	"file-management-service/pkg/validator"
)

type AuditHandler struct {
	auditRepo domrepo.AuditRepository
	validator *validator.Validator
}

func NewAuditHandler(auditRepo domrepo.AuditRepository, v *validator.Validator) *AuditHandler {
	return &AuditHandler{auditRepo: auditRepo, validator: v}
}

// List godoc
// @Summary      List audit logs
// @Description  Get paginated audit logs (admin only)
// @Tags         audit
// @Produce      json
// @Security     BearerAuth
// @Param        page       query  int     false  "Page" default(1)
// @Param        page_size  query  int     false  "Page size" default(20)
// @Param        user_id    query  string  false  "Filter by user ID"
// @Param        action     query  string  false  "Filter by action"
// @Success      200  {object}  response.Response
// @Router       /audit-logs [get]
func (h *AuditHandler) List(c *fiber.Ctx) error {
	pag, err := pagination.ParseFromQuery(c)
	if err != nil {
		return response.BadRequest(c, "invalid pagination")
	}

	filter := domrepo.AuditFilter{
		Page:      pag.Page,
		PageSize:  pag.PageSize,
		OrderBy:   c.Query("order_by", "created_at"),
		OrderDir:  c.Query("order_dir", "desc"),
		Status:    c.Query("status"),
		IPAddress: c.Query("ip_address"),
	}

	if action := c.Query("action"); action != "" {
		a := entity.AuditAction(action)
		filter.Action = &a
	}
	if resType := c.Query("resource_type"); resType != "" {
		rt := resType
		filter.ResourceType = &rt
	}
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		uid, err := uuid.Parse(userIDStr)
		if err != nil {
			return response.BadRequest(c, "invalid user_id UUID")
		}
		filter.UserID = &uid
	}
	if resIDStr := c.Query("resource_id"); resIDStr != "" {
		rid, err := uuid.Parse(resIDStr)
		if err != nil {
			return response.BadRequest(c, "invalid resource_id UUID")
		}
		filter.ResourceID = &rid
	}
	if start := c.Query("start_date"); start != "" {
		t, err := time.Parse(time.RFC3339, start)
		if err != nil {
			return response.BadRequest(c, "start_date must be RFC3339 format")
		}
		filter.StartDate = &t
	}
	if end := c.Query("end_date"); end != "" {
		t, err := time.Parse(time.RFC3339, end)
		if err != nil {
			return response.BadRequest(c, "end_date must be RFC3339 format")
		}
		filter.EndDate = &t
	}

	logs, total, err := h.auditRepo.List(c.Context(), filter)
	if err != nil {
		return handleError(c, err)
	}

	meta := pagination.NewMeta(pag.Page, pag.PageSize, total)
	return response.SuccessWithMeta(c, fiber.StatusOK, "audit logs retrieved", logs, meta)
}

// GetByID godoc
// @Summary      Get audit log entry
// @Description  Get a specific audit log entry by ID (admin only)
// @Tags         audit
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Audit log ID (UUID)"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /audit-logs/{id} [get]
func (h *AuditHandler) GetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return response.BadRequest(c, "invalid UUID")
	}

	log, err := h.auditRepo.GetByID(c.Context(), id)
	if err != nil {
		return handleError(c, err)
	}
	return response.OK(c, "audit log retrieved", log)
}

// Export godoc
// @Summary      Export audit logs
// @Description  Export audit logs as CSV (admin only)
// @Tags         audit
// @Produce      text/csv
// @Security     BearerAuth
// @Success      200
// @Router       /audit-logs/export [get]
func (h *AuditHandler) Export(c *fiber.Ctx) error {
	filter := domrepo.AuditFilter{
		Page:     1,
		PageSize: 10000,
		OrderBy:  "created_at",
		OrderDir: "desc",
	}

	logs, _, err := h.auditRepo.List(c.Context(), filter)
	if err != nil {
		return handleError(c, err)
	}

	var buf bytes.Buffer
	csvW := csv.NewWriter(&buf)

	_ = csvW.Write([]string{
		"ID", "UserID", "Action", "ResourceID", "ResourceType",
		"IPAddress", "Status", "CreatedAt",
	})
	for _, l := range logs {
		userID := ""
		if l.UserID != nil {
			userID = l.UserID.String()
		}
		resID := ""
		if l.ResourceID != nil {
			resID = l.ResourceID.String()
		}
		resType := ""
		if l.ResourceType != nil {
			resType = *l.ResourceType
		}
		_ = csvW.Write([]string{
			l.ID.String(), userID, string(l.Action), resID, resType,
			l.IPAddress, l.Status, l.CreatedAt.Format(time.RFC3339),
		})
	}
	csvW.Flush()

	filename := fmt.Sprintf("audit_logs_%s.csv", time.Now().Format("20060102_150405"))
	c.Set(fiber.HeaderContentType, "text/csv")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.Send(buf.Bytes())
}

