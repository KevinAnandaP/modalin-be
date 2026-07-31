package handler

import (
	"encoding/json"
	"log"

	paymentService "modalin-be/internal/payment/service"
	"modalin-be/pkg/config"
	"modalin-be/pkg/xendit"

	"github.com/gofiber/fiber/v2"
)

type WebhookHandler struct {
	xenditSvc    paymentService.XenditPaymentService
	xenditClient *xendit.Client
}

func NewWebhookHandler(xenditSvc paymentService.XenditPaymentService, xenditClient *xendit.Client) *WebhookHandler {
	return &WebhookHandler{
		xenditSvc:    xenditSvc,
		xenditClient: xenditClient,
	}
}

// HandleXenditCallback handles incoming HTTP POST requests from Xendit Webhooks
func (h *WebhookHandler) HandleXenditCallback(c *fiber.Ctx) error {
	// Verify Xendit Webhook Verification Token Header
	callbackToken := c.Get("x-callback-token")
	if config.AppConfig.XenditWebhookToken != "" && callbackToken != config.AppConfig.XenditWebhookToken {
		log.Printf("[Xendit Webhook Warning] Invalid callback token header: %s", callbackToken)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized webhook token",
		})
	}

	bodyBytes := c.Body()

	// Try unmarshalling into InvoiceWebhookPayload
	var invoicePayload xendit.InvoiceWebhookPayload
	if err := json.Unmarshal(bodyBytes, &invoicePayload); err == nil && invoicePayload.ExternalID != "" && invoicePayload.Status != "" {
		if err := h.xenditSvc.ProcessInvoiceCallback(c.Context(), invoicePayload); err != nil {
			log.Printf("[Xendit Webhook Error] Failed to process invoice callback: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "success",
			"type":   "invoice",
		})
	}

	// Try unmarshalling into DisbursementWebhookPayload
	var disbPayload xendit.DisbursementWebhookPayload
	if err := json.Unmarshal(bodyBytes, &disbPayload); err == nil && disbPayload.ExternalID != "" && disbPayload.Status != "" {
		if err := h.xenditSvc.ProcessDisbursementCallback(c.Context(), disbPayload); err != nil {
			log.Printf("[Xendit Webhook Error] Failed to process disbursement callback: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "success",
			"type":   "disbursement",
		})
	}

	log.Printf("[Xendit Webhook] Unrecognized webhook body: %s", string(bodyBytes))
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Webhook received but ignored",
	})
}
