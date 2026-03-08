package middleware

import (
	"github.com/gofiber/fiber/v2"

	"file-management-service/internal/domain/entity"
	"file-management-service/pkg/response"
)

// RBACMiddleware enforces role-based access control.
type RBACMiddleware struct{}

// NewRBACMiddleware creates a new RBACMiddleware.
func NewRBACMiddleware() *RBACMiddleware {
	return &RBACMiddleware{}
}

// RequireRole ensures the authenticated user has one of the listed roles.
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

// RequireMinRole ensures the user's role level is at least minRole.
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

// RequireSuperAdmin allows only super_admin users.
func (m *RBACMiddleware) RequireSuperAdmin() fiber.Handler {
	return m.RequireRole(entity.RoleSuperAdmin)
}

// RequireAdmin allows admin and super_admin users.
func (m *RBACMiddleware) RequireAdmin() fiber.Handler {
	return m.RequireMinRole(entity.RoleAdmin)
}

// RequireManager allows manager and above.
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
