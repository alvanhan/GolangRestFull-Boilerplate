package pagination

import (
	"file-management-service/pkg/response"

	"github.com/gofiber/fiber/v2"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// Pagination holds paging parameters parsed from a query string.
type Pagination struct {
	Page     int `query:"page"      validate:"min=1"`
	PageSize int `query:"page_size" validate:"min=1,max=100"`
}

// Offset returns the zero-based row offset for SQL OFFSET clauses.
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Normalize applies default values when Page or PageSize are zero,
// and caps PageSize at maxPageSize.
func (p *Pagination) Normalize() {
	if p.Page <= 0 {
		p.Page = defaultPage
	}
	if p.PageSize <= 0 {
		p.PageSize = defaultPageSize
	}
	if p.PageSize > maxPageSize {
		p.PageSize = maxPageSize
	}
}

// NewMeta builds a response.Meta value from paging parameters and the total
// item count.
func NewMeta(page, pageSize int, total int64) *response.Meta {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return &response.Meta{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}
}

// ParseFromQuery parses Pagination from a Fiber request's query parameters and
// immediately normalizes defaults.
func ParseFromQuery(c *fiber.Ctx) (*Pagination, error) {
	p := &Pagination{}
	if err := c.QueryParser(p); err != nil {
		return nil, err
	}
	p.Normalize()
	return p, nil
}
