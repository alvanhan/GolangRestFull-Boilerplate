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

type AuthMiddleware struct {
	jwtService pkgjwt.JWTService
}

func NewAuthMiddleware(jwtService pkgjwt.JWTService) *AuthMiddleware {
	return &AuthMiddleware{jwtService: jwtService}
}

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

func GetUserID(c *fiber.Ctx) string {
	id, _ := c.Locals(ContextKeyUserID).(string)
	return id
}

func GetUserRole(c *fiber.Ctx) string {
	role, _ := c.Locals(ContextKeyRole).(string)
	return role
}

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

