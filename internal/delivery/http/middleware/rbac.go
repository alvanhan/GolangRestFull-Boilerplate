package middleware

import (
	"github.com/gofiber/fiber/v2"

	"file-management-service/internal/domain/entity"
	"file-management-service/pkg/response"
)

type RBACMiddleware struct{}

func NewRBACMiddleware() *RBACMiddleware {
	return &RBACMiddleware{}
}

func (m *RBACMiddleware) RequireRole(roles ...entity.UserRole) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := GetUserRole(c)
		if userRole == "" {
			return response.Unauthorized(c, "authentication required")
		}
		for _, r := range roles {
			if entity.UserRole(userRole) == r {
				return c.Next()
			}
		}
		return response.Forbidden(c, "insufficient permissions")
	}
}

func (m *RBACMiddleware) RequireMinRole(minRole entity.UserRole) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := GetUserRole(c)
		if userRole == "" {
			return response.Unauthorized(c, "authentication required")
		}
		if roleLevel(entity.UserRole(userRole)) < roleLevel(minRole) {
			return response.Forbidden(c, "insufficient role level")
		}
		return c.Next()
	}
}

func (m *RBACMiddleware) RequireSuperAdmin() fiber.Handler {
	return m.RequireRole(entity.RoleSuperAdmin)
}

func (m *RBACMiddleware) RequireAdmin() fiber.Handler {
	return m.RequireMinRole(entity.RoleAdmin)
}

func (m *RBACMiddleware) RequireManager() fiber.Handler {
	return m.RequireMinRole(entity.RoleManager)
}

func roleLevel(role entity.UserRole) int {
	switch role {
	case entity.RoleSuperAdmin:
		return 5
	case entity.RoleAdmin:
		return 4
	case entity.RoleManager:
		return 3
	case entity.RoleEditor:
		return 2
	case entity.RoleViewer:
		return 1
	default:
		return 0
	}
}
