package handler

import (
	"errors"

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
	user, err := h.service.Register(c.Context(), service.RegisterInput{FullName: request.FullName, Email: request.Email, Phone: request.Phone, Password: request.Password, City: request.City, Address: request.Address, TermsAccepted: request.TermsAccepted})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTermsNotAccepted):
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrEmailTaken):
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(400).JSON(fiber.Map{"error": "registration failed"})
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
	var request struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&request); err != nil || request.Role == "" {
		return c.Status(400).JSON(fiber.Map{"error": "role is required"})
	}
	roleRequest, err := h.service.RequestRole(c.Context(), userID, request.Role)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserInactive):
			return c.Status(403).JSON(fiber.Map{"error": "account is not active"})
		case errors.Is(err, service.ErrInvalidRole):
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrRoleRequestExists):
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(500).JSON(fiber.Map{"error": "failed to request role"})
		}
	}
	return c.Status(201).JSON(fiber.Map{"data": roleRequest})
}
