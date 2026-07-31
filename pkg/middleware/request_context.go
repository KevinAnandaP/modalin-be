package middleware

import (
	"modalin-be/pkg/audit"

	"github.com/gofiber/fiber/v2"
)

// PropagateRequestID makes the reverse-proxy/Fiber request ID available to
// service-layer context.Context values used when persisting audit events.
func PropagateRequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID, _ := c.Locals("requestid").(string)
		c.Context().SetUserValue("modalin.request_id", requestID)
		c.SetUserContext(audit.WithRequestID(c.UserContext(), requestID))
		return c.Next()
	}
}
