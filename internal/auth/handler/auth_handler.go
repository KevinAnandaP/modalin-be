package handler

import (
	"errors"
	"strings"

	"modalin-be/internal/auth/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthHandler struct{ service *service.AuthService }

func NewAuthHandler(service *service.AuthService) *AuthHandler { return &AuthHandler{service: service} }

type registerRequest struct {
	FullName      string `json:"full_name" validate:"required"`
	Email         string `json:"email" validate:"required,email"`
	Phone         string `json:"phone" validate:"required"`
	Password      string `json:"password" validate:"required"`
	City          string `json:"city" validate:"required"`
	Address       string `json:"address" validate:"required"`
	TermsAccepted bool   `json:"terms_accepted"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var request registerRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	if request.FullName == "" || request.Email == "" || request.Phone == "" || request.Password == "" || request.City == "" || request.Address == "" {
		return c.Status(400).JSON(fiber.Map{"error": "all profile fields are required"})
	}
	user, err := h.service.Register(c.Context(), service.RegisterInput{
		FullName:      request.FullName,
		Email:         request.Email,
		Phone:         request.Phone,
		Password:      request.Password,
		City:          request.City,
		Address:       request.Address,
		TermsAccepted: request.TermsAccepted,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTermsNotAccepted), errors.Is(err, service.ErrPasswordLength):
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrEmailTaken):
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
	}
	return c.Status(201).JSON(fiber.Map{"data": user})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&request); err != nil || request.Email == "" || request.Password == "" {
		return c.Status(400).JSON(fiber.Map{"error": "email and password are required"})
	}
	result, err := h.service.Login(c.Context(), request.Email, request.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserInactive) {
			return c.Status(403).JSON(fiber.Map{"error": "account is not active"})
		}
		return c.Status(401).JSON(fiber.Map{"error": "invalid email or password"})
	}
	return c.JSON(fiber.Map{"data": result})
}

func (h *AuthHandler) GoogleAuth(c *fiber.Ctx) error {
	var req struct {
		IDToken    string `json:"id_token"`
		Credential string `json:"credential"`
		Token      string `json:"token"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}
	token := strings.TrimSpace(req.IDToken)
	if token == "" {
		token = strings.TrimSpace(req.Credential)
	}
	if token == "" {
		token = strings.TrimSpace(req.Token)
	}
	if token == "" {
		return c.Status(400).JSON(fiber.Map{"error": "id_token is required"})
	}
	res, err := h.service.GoogleAuth(c.Context(), token)
	if err != nil {
		if errors.Is(err, service.ErrUserInactive) {
			return c.Status(403).JSON(fiber.Map{"error": "account is not active"})
		}
		if errors.Is(err, service.ErrGoogleAccountLinkRequired) {
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"data": res})
}

func (h *AuthHandler) CompleteGoogleAuth(c *fiber.Ctx) error {
	var req service.CompleteGoogleRegistrationInput
	if err := c.BodyParser(&req); err != nil || req.TempToken == "" {
		return c.Status(400).JSON(fiber.Map{"error": "temp_token, phone, city, and address are required"})
	}
	res, err := h.service.CompleteGoogleRegistration(c.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTermsNotAccepted), errors.Is(err, service.ErrInvalidTempToken):
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrEmailTaken):
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
	}
	return c.Status(201).JSON(fiber.Map{"data": res})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}
	profile, err := h.service.Profile(c.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserInactive) {
			return c.Status(403).JSON(fiber.Map{"error": "account is not active"})
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(401).JSON(fiber.Map{"error": "user not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to get profile"})
	}
	return c.JSON(fiber.Map{"data": profile})
}

func (h *AuthHandler) RequestRole(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}
	var request service.RequestRoleInput
	if err := c.BodyParser(&request); err != nil || request.Role == "" {
		return c.Status(400).JSON(fiber.Map{"error": "role is required"})
	}
	roleRequest, err := h.service.RequestRole(c.Context(), userID, request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserInactive):
			return c.Status(403).JSON(fiber.Map{"error": "account is not active"})
		case errors.Is(err, service.ErrInvalidRole),
			errors.Is(err, service.ErrIdentityCardRequired),
			errors.Is(err, service.ErrRiskAgreementRequired),
			errors.Is(err, service.ErrEthicsAgreementRequired),
			errors.Is(err, service.ErrTrainingRequired):
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrRoleRequestExists):
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(500).JSON(fiber.Map{"error": "failed to request role"})
		}
	}
	return c.Status(201).JSON(fiber.Map{"data": roleRequest})
}

func (h *AuthHandler) GetRoleRequests(c *fiber.Ctx) error {
	status := c.Query("status", "")
	requests, err := h.service.GetRoleRequests(c.Context(), status)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to fetch role requests"})
	}
	return c.JSON(fiber.Map{"data": requests})
}

func (h *AuthHandler) ReviewRoleRequest(c *fiber.Ctx) error {
	adminID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(401).JSON(fiber.Map{"error": "invalid authorization context"})
	}
	var request service.ReviewRoleRequestInput
	if err := c.BodyParser(&request); err != nil || request.RequestID == uuid.Nil || request.Action == "" {
		return c.Status(400).JSON(fiber.Map{"error": "request_id and action are required"})
	}
	reviewed, err := h.service.ReviewRoleRequest(c.Context(), adminID, request)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRoleRequestNotFound):
			return c.Status(444).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidAction):
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(500).JSON(fiber.Map{"error": "failed to review role request"})
		}
	}
	return c.JSON(fiber.Map{"data": reviewed})
}
