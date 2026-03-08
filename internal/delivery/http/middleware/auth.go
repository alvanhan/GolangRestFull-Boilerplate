package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	pkgjwt "file-management-service/pkg/jwt"
	"file-management-service/pkg/response"
)

const (
	ContextKeyUserID = "user_id"
	ContextKeyEmail  = "email"
	ContextKeyRole   = "role"
	ContextKeyClaims = "claims"
)

// AuthMiddleware validates JWT tokens on protected routes.
type AuthMiddleware struct {
	jwtService pkgjwt.JWTService
}

// NewAuthMiddleware creates a new AuthMiddleware.
func NewAuthMiddleware(jwtService pkgjwt.JWTService) *AuthMiddleware {
	return &AuthMiddleware{jwtService: jwtService}
}

// Authenticate is a Fiber middleware that validates the Bearer JWT token.
func (m *AuthMiddleware) Authenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractBearerToken(c)
		if token == "" {
			return response.Unauthorized(c, "missing or malformed authorization header")
		}

		claims, err := m.jwtService.ValidateToken(token, pkgjwt.AccessToken)
		if err != nil {
			return response.Unauthorized(c, "invalid or expired token")
		}

		c.Locals(ContextKeyUserID, claims.UserID.String())
		c.Locals(ContextKeyEmail, claims.Email)
		c.Locals(ContextKeyRole, claims.Role)
		c.Locals(ContextKeyClaims, claims)

		return c.Next()
	}
}

// OptionalAuthenticate parses the token if present but does not fail if absent.
func (m *AuthMiddleware) OptionalAuthenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractBearerToken(c)
		if token == "" {
			return c.Next()
		}

		claims, err := m.jwtService.ValidateToken(token, pkgjwt.AccessToken)
		if err != nil {
			return c.Next()
		}

		c.Locals(ContextKeyUserID, claims.UserID.String())
		c.Locals(ContextKeyEmail, claims.Email)
		c.Locals(ContextKeyRole, claims.Role)
		c.Locals(ContextKeyClaims, claims)

		return c.Next()
	}
}

// GetUserID retrieves the authenticated user ID from fiber context.
func GetUserID(c *fiber.Ctx) string {
	id, _ := c.Locals(ContextKeyUserID).(string)
	return id
}

// GetUserRole retrieves the authenticated user role from fiber context.
func GetUserRole(c *fiber.Ctx) string {
	role, _ := c.Locals(ContextKeyRole).(string)
	return role
}

// GetUserEmail retrieves the authenticated user email from fiber context.
func GetUserEmail(c *fiber.Ctx) string {
	email, _ := c.Locals(ContextKeyEmail).(string)
	return email
}

func extractBearerToken(c *fiber.Ctx) string {
	header := c.Get(fiber.HeaderAuthorization)
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}

