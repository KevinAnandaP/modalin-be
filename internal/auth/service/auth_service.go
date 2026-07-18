package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"modalin-be/internal/auth/repository"
	"modalin-be/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrTermsNotAccepted       = errors.New("terms must be accepted")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrPasswordLength         = errors.New("password must be between 8 and 72 characters")
	ErrUserInactive           = errors.New("user is not active")
	ErrEmailTaken             = errors.New("email already registered")
	ErrInvalidRole            = errors.New("role cannot be requested")
	ErrRoleRequestExists      = errors.New("role request already open")
	ErrIdentityCardRequired   = errors.New("identity card photo is required for this role")
	ErrRiskAgreementRequired = errors.New("risk agreement must be accepted for lender role")
	ErrEthicsAgreementRequired = errors.New("ethics agreement must be accepted for verifier role")
	ErrTrainingRequired       = errors.New("mini-training must be completed for verifier role")
	ErrRoleRequestNotFound    = errors.New("role request not found")
	ErrInvalidAction          = errors.New("invalid review action")
)

type RegisterInput struct {
	FullName, Email, Phone, Password, City, Address string
	TermsAccepted                                   bool
}

type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Profile struct {
	User  *model.User `json:"user"`
	Roles []string    `json:"roles"`
}

type RequestRoleInput struct {
	Role                  string  `json:"role"`
	IdentityCardURL       *string `json:"identity_card_url"`
	RiskAgreementAccepted bool    `json:"risk_agreement_accepted"`
	EthicsAccepted        bool    `json:"ethics_accepted"`
	TrainingCompleted     bool    `json:"training_completed"`
}

type ReviewRoleRequestInput struct {
	RequestID uuid.UUID `json:"request_id"`
	Action    string    `json:"action"` // approve, reject, revision_required
	AdminNote string    `json:"admin_note"`
}

type AuthService struct {
	repo   repository.Repository
	secret []byte
}

func NewAuthService(repo repository.Repository, secret string) *AuthService {
	return &AuthService{repo: repo, secret: []byte(secret)}
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*model.User, error) {
	if !in.TermsAccepted {
		return nil, ErrTermsNotAccepted
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if len(in.Password) > 72 || len(in.Password) < 8 {
		return nil, ErrPasswordLength
	}
	_, err := s.repo.FindUserByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailTaken
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	user := &model.User{
		FullName:        strings.TrimSpace(in.FullName),
		Email:           email,
		Phone:           strings.TrimSpace(in.Phone),
		PasswordHash:    string(hash),
		City:            strings.TrimSpace(in.City),
		Address:         strings.TrimSpace(in.Address),
		Status:          "active",
		TermsAcceptedAt: &now,
		TermsVersion:    "v1",
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := s.repo.FindUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	if user.Status != "active" {
		return nil, ErrUserInactive
	}
	roles, err := s.repo.GetApprovedRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return s.issueToken(user.ID, roles)
}

func (s *AuthService) Profile(ctx context.Context, id uuid.UUID) (*Profile, error) {
	user, err := s.repo.FindUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user.Status != "active" {
		return nil, ErrUserInactive
	}
	roles, err := s.repo.GetApprovedRoles(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Profile{User: user, Roles: roles}, nil
}

func (s *AuthService) RequestRole(ctx context.Context, userID uuid.UUID, in RequestRoleInput) (*model.RoleRequest, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Status != "active" {
		return nil, ErrUserInactive
	}

	roleName := strings.ToLower(strings.TrimSpace(in.Role))
	if roleName == "admin" || (roleName != "borrower" && roleName != "lender" && roleName != "verifier") {
		return nil, ErrInvalidRole
	}

	// Validate role-specific requirements
	if roleName == "lender" {
		if in.IdentityCardURL == nil || strings.TrimSpace(*in.IdentityCardURL) == "" {
			return nil, ErrIdentityCardRequired
		}
		if !in.RiskAgreementAccepted {
			return nil, ErrRiskAgreementRequired
		}
	} else if roleName == "verifier" {
		if in.IdentityCardURL == nil || strings.TrimSpace(*in.IdentityCardURL) == "" {
			return nil, ErrIdentityCardRequired
		}
		if !in.EthicsAccepted {
			return nil, ErrEthicsAgreementRequired
		}
		if !in.TrainingCompleted {
			return nil, ErrTrainingRequired
		}
	}

	role, err := s.repo.FindRoleByName(ctx, roleName)
	if err != nil {
		return nil, ErrInvalidRole
	}

	exists, err := s.repo.HasOpenRoleRequest(ctx, userID, role.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrRoleRequestExists
	}

	request := &model.RoleRequest{
		UserID:                userID,
		RoleID:                role.ID,
		Status:                "submitted",
		IdentityCardURL:       in.IdentityCardURL,
		RiskAgreementAccepted: in.RiskAgreementAccepted,
		EthicsAccepted:        in.EthicsAccepted,
		TrainingCompleted:     in.TrainingCompleted,
	}

	if err := s.repo.CreateRoleRequest(ctx, request); err != nil {
		return nil, err
	}
	return request, nil
}

func (s *AuthService) GetRoleRequests(ctx context.Context, status string) ([]model.RoleRequest, error) {
	return s.repo.GetRoleRequests(ctx, status)
}

func (s *AuthService) ReviewRoleRequest(ctx context.Context, adminID uuid.UUID, in ReviewRoleRequestInput) (*model.RoleRequest, error) {
	req, err := s.repo.FindRoleRequestByID(ctx, in.RequestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleRequestNotFound
		}
		return nil, err
	}

	action := strings.ToLower(strings.TrimSpace(in.Action))
	now := time.Now().UTC()

	switch action {
	case "approve":
		req.Status = "approved"
		req.ApprovedBy = &adminID
		req.ApprovedAt = &now
		if err := s.repo.UpdateRoleRequest(ctx, req); err != nil {
			return nil, err
		}
		if err := s.repo.ApproveUserRole(ctx, req.UserID, req.RoleID, adminID); err != nil {
			return nil, err
		}
	case "reject":
		req.Status = "rejected"
		req.AdminNote = &in.AdminNote
		req.ApprovedBy = &adminID
		req.ApprovedAt = &now
		if err := s.repo.UpdateRoleRequest(ctx, req); err != nil {
			return nil, err
		}
	case "revision_required":
		req.Status = "revision_required"
		req.AdminNote = &in.AdminNote
		if err := s.repo.UpdateRoleRequest(ctx, req); err != nil {
			return nil, err
		}
	default:
		return nil, ErrInvalidAction
	}

	return req, nil
}

func (s *AuthService) issueToken(userID uuid.UUID, roles []string) (*LoginResult, error) {
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	claims := jwt.MapClaims{"user_id": userID.String(), "roles": roles, "iss": "modalin-be", "iat": now.Unix(), "exp": expires.Unix()}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, ExpiresAt: expires}, nil
}
