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
	ErrTermsNotAccepted        = errors.New("terms must be accepted")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrPasswordLength          = errors.New("password must be between 8 and 72 characters")
	ErrUserInactive            = errors.New("user is not active")
	ErrEmailTaken              = errors.New("email already registered")
	ErrInvalidRole             = errors.New("role cannot be requested")
	ErrRoleRequestExists       = errors.New("role request already open")
	ErrIdentityCardRequired    = errors.New("identity card photo is required for this role")
	ErrRiskAgreementRequired   = errors.New("risk agreement must be accepted for lender role")
	ErrEthicsAgreementRequired = errors.New("ethics agreement must be accepted for verifier role")
	ErrTrainingRequired        = errors.New("mini-training must be completed for verifier role")
	ErrRoleRequestNotFound     = errors.New("role request not found")
	ErrInvalidAction           = errors.New("invalid review action")
	ErrInvalidTempToken        = errors.New("invalid or expired temporary registration token")
	ErrGoogleAuthFailed        = errors.New("invalid google auth credentials")
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

type GooglePrefill struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	GoogleID string `json:"google_id"`
}

type GoogleAuthResult struct {
	IsNewUser   bool           `json:"is_new_user"`
	Prefill     *GooglePrefill `json:"prefill,omitempty"`
	TempToken   string         `json:"temp_token,omitempty"`
	LoginResult *LoginResult   `json:"login_result,omitempty"`
}

type CompleteGoogleRegistrationInput struct {
	TempToken     string `json:"temp_token"`
	Phone         string `json:"phone"`
	City          string `json:"city"`
	Address       string `json:"address"`
	TermsAccepted bool   `json:"terms_accepted"`
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

func (s *AuthService) GoogleAuth(ctx context.Context, googleID, email, fullName string) (*GoogleAuthResult, error) {
	googleID = strings.TrimSpace(googleID)
	email = strings.ToLower(strings.TrimSpace(email))
	fullName = strings.TrimSpace(fullName)

	if googleID == "" || email == "" {
		return nil, ErrGoogleAuthFailed
	}

	// 1. Try finding user by GoogleID
	user, err := s.repo.FindUserByGoogleID(ctx, googleID)
	if err == nil {
		if user.Status != "active" {
			return nil, ErrUserInactive
		}
		roles, err := s.repo.GetApprovedRoles(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		loginRes, err := s.issueToken(user.ID, roles)
		if err != nil {
			return nil, err
		}
		return &GoogleAuthResult{IsNewUser: false, LoginResult: loginRes}, nil
	}

	// 2. Try finding user by Email
	user, err = s.repo.FindUserByEmail(ctx, email)
	if err == nil {
		if user.Status != "active" {
			return nil, ErrUserInactive
		}
		user.GoogleID = &googleID
		_ = s.repo.CreateUser(ctx, user)
		roles, err := s.repo.GetApprovedRoles(ctx, user.ID)
		if err != nil {
			return nil, err
		}
		loginRes, err := s.issueToken(user.ID, roles)
		if err != nil {
			return nil, err
		}
		return &GoogleAuthResult{IsNewUser: false, LoginResult: loginRes}, nil
	}

	// 3. New User - Issue temporary token for 2-step onboarding
	tempToken, err := s.issueTempGoogleToken(googleID, email, fullName)
	if err != nil {
		return nil, err
	}

	return &GoogleAuthResult{
		IsNewUser: true,
		Prefill: &GooglePrefill{
			Email:    email,
			FullName: fullName,
			GoogleID: googleID,
		},
		TempToken: tempToken,
	}, nil
}

func (s *AuthService) CompleteGoogleRegistration(ctx context.Context, in CompleteGoogleRegistrationInput) (*LoginResult, error) {
	if !in.TermsAccepted {
		return nil, ErrTermsNotAccepted
	}
	if strings.TrimSpace(in.Phone) == "" || strings.TrimSpace(in.City) == "" || strings.TrimSpace(in.Address) == "" {
		return nil, errors.New("phone, city, and address are required to complete registration")
	}

	prefill, err := s.parseTempGoogleToken(in.TempToken)
	if err != nil {
		return nil, ErrInvalidTempToken
	}

	_, err = s.repo.FindUserByEmail(ctx, prefill.Email)
	if err == nil {
		return nil, ErrEmailTaken
	}

	randomPassword := uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user := &model.User{
		FullName:        prefill.FullName,
		Email:           prefill.Email,
		Phone:           strings.TrimSpace(in.Phone),
		PasswordHash:    string(hash),
		City:            strings.TrimSpace(in.City),
		Address:         strings.TrimSpace(in.Address),
		Status:          "active",
		GoogleID:        &prefill.GoogleID,
		TermsAcceptedAt: &now,
		TermsVersion:    "v1",
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
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

func (s *AuthService) issueTempGoogleToken(googleID, email, fullName string) (string, error) {
	now := time.Now().UTC()
	expires := now.Add(30 * time.Minute)
	claims := jwt.MapClaims{
		"google_id": googleID,
		"email":     email,
		"full_name": fullName,
		"iss":       "modalin-be-google-temp",
		"iat":       now.Unix(),
		"exp":       expires.Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

func (s *AuthService) parseTempGoogleToken(tempToken string) (*GooglePrefill, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tempToken, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidTempToken
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid || claims["iss"] != "modalin-be-google-temp" {
		return nil, ErrInvalidTempToken
	}

	googleID, _ := claims["google_id"].(string)
	email, _ := claims["email"].(string)
	fullName, _ := claims["full_name"].(string)

	if googleID == "" || email == "" {
		return nil, ErrInvalidTempToken
	}

	return &GooglePrefill{
		GoogleID: googleID,
		Email:    email,
		FullName: fullName,
	}, nil
}
