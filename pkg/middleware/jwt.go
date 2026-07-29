package middleware

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuthorizationStateReader supplies the current persisted account state. When
// configured, JWT roles are never trusted over the database's current roles.
type AuthorizationStateReader interface {
	GetAuthorizationState(context.Context, uuid.UUID) (status string, tokenVersion uint64, roles []string, err error)
}

func JWTProtected(secret string, readers ...AuthorizationStateReader) fiber.Handler {
	var reader AuthorizationStateReader
	if len(readers) > 0 {
		reader = readers[0]
	}
	return func(c *fiber.Ctx) error {
		header := c.Get(fiber.HeaderAuthorization)
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or invalid authorization token"})
		}
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid || claims["iss"] != "modalin-be" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired authorization token"})
		}
		idString, ok := claims["user_id"].(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization token"})
		}
		userID, err := uuid.Parse(idString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization token"})
		}
		roles := make([]string, 0)
		if rawRoles, ok := claims["roles"].([]interface{}); ok {
			for _, role := range rawRoles {
				if name, ok := role.(string); ok {
					roles = append(roles, name)
				}
			}
		}
		if reader != nil {
			claimVersion, ok := tokenVersion(claims["token_version"])
			if !ok {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization token"})
			}
			status, currentVersion, currentRoles, err := reader.GetAuthorizationState(c.Context(), userID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization token"})
				}
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "authorization service unavailable"})
			}
			if !strings.EqualFold(status, "active") {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account is not active"})
			}
			if claimVersion != currentVersion {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authorization token has been revoked"})
			}
			roles = currentRoles
		}
		c.Locals("user_id", userID)
		c.Locals("roles", roles)
		return c.Next()
	}
}

func tokenVersion(value any) (uint64, bool) {
	number, ok := value.(float64)
	if !ok || number < 1 || number != math.Trunc(number) || number > math.MaxUint64 {
		return 0, false
	}
	return uint64(number), true
}

func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roles, ok := c.Locals("roles").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient role"})
		}
		for _, role := range roles {
			for _, allowed := range allowedRoles {
				if role == allowed {
					return c.Next()
				}
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "insufficient role"})
	}
}
