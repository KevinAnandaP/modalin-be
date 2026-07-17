package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"modalin-be/internal/auth/service"
	"modalin-be/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type fakeRepository struct {
	user            *model.User
	roles           []string
	role            *model.Role
	createdUser     *model.User
	createdRequest  *model.RoleRequest
	openRoleRequest bool
}

func (r *fakeRepository) CreateUser(_ context.Context, user *model.User) error {
	r.createdUser = user
	user.ID = uuid.New()
	return nil
}
func (r *fakeRepository) FindUserByEmail(_ context.Context, email string) (*model.User, error) {
	if r.user != nil && r.user.Email == email {
		return r.user, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeRepository) FindUserByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if r.user != nil && r.user.ID == id {
		return r.user, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeRepository) FindRoleByName(_ context.Context, _ string) (*model.Role, error) {
	if r.role == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return r.role, nil
}
func (r *fakeRepository) CreateRoleRequest(_ context.Context, request *model.RoleRequest) error {
	r.createdRequest = request
	return nil
}
func (r *fakeRepository) HasOpenRoleRequest(_ context.Context, _ uuid.UUID, _ int) (bool, error) {
	return r.openRoleRequest, nil
}
func (r *fakeRepository) GetApprovedRoles(_ context.Context, _ uuid.UUID) ([]string, error) {
	return r.roles, nil
}

func TestRegisterRejectsUnacceptedTerms(t *testing.T) {
	svc := service.NewAuthService(nil, "test-secret")

	_, err := svc.Register(context.Background(), service.RegisterInput{
		FullName:      "Budi Santoso",
		Email:         "budi@example.com",
		Phone:         "08123456789",
		Password:      strings.Repeat("x", 12),
		City:          "Jakarta",
		Address:       "Jl. Merdeka 1",
		TermsAccepted: false,
	})

	if err != service.ErrTermsNotAccepted {
		t.Fatalf("expected ErrTermsNotAccepted, got %v", err)
	}
}

func TestRegisterHashesPasswordAndNormalizesEmail(t *testing.T) {
	repo := &fakeRepository{}
	svc := service.NewAuthService(repo, "test-secret")
	password := strings.Repeat("x", 12)
	user, err := svc.Register(context.Background(), service.RegisterInput{FullName: "Budi", Email: " BUDI@EXAMPLE.COM ", Phone: "0812", Password: password, City: "Jakarta", Address: "Alamat", TermsAccepted: true})
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "budi@example.com" || repo.createdUser == nil {
		t.Fatalf("user was not normalized and saved: %#v", user)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		t.Fatal("password was not hashed correctly")
	}
}

func TestLoginIssuesTokenWithApprovedRoles(t *testing.T) {
	password := strings.Repeat("x", 12)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := &model.User{ID: uuid.New(), Email: "budi@example.com", PasswordHash: string(hash), Status: "active"}
	svc := service.NewAuthService(&fakeRepository{user: user, roles: []string{"lender"}}, "test-secret")
	result, err := svc.Login(context.Background(), user.Email, password)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := jwt.Parse(result.Token, func(token *jwt.Token) (interface{}, error) { return []byte("test-secret"), nil })
	if err != nil || !parsed.Valid {
		t.Fatalf("token is invalid: %v", err)
	}
	roles := parsed.Claims.(jwt.MapClaims)["roles"].([]interface{})
	if len(roles) != 1 || roles[0] != "lender" {
		t.Fatalf("unexpected roles: %#v", roles)
	}
}

func TestRequestRoleRejectsAdmin(t *testing.T) {
	svc := service.NewAuthService(&fakeRepository{}, "test-secret")
	_, err := svc.RequestRole(context.Background(), uuid.New(), "admin")
	if !errors.Is(err, service.ErrInvalidRole) {
		t.Fatalf("expected ErrInvalidRole, got %v", err)
	}
}

func TestRequestRoleCreatesBorrowerRequest(t *testing.T) {
	repo := &fakeRepository{role: &model.Role{ID: 1, Name: "borrower"}}
	svc := service.NewAuthService(repo, "test-secret")
	userID := uuid.New()
	request, err := svc.RequestRole(context.Background(), userID, " borrower ")
	if err != nil {
		t.Fatal(err)
	}
	if request.Status != "submitted" || repo.createdRequest.UserID != userID || repo.createdRequest.RoleID != 1 {
		t.Fatalf("unexpected request: %#v", repo.createdRequest)
	}
}

func TestRequestRoleRejectsOpenDuplicate(t *testing.T) {
	svc := service.NewAuthService(&fakeRepository{role: &model.Role{ID: 1}, openRoleRequest: true}, "test-secret")
	_, err := svc.RequestRole(context.Background(), uuid.New(), "borrower")
	if !errors.Is(err, service.ErrRoleRequestExists) {
		t.Fatalf("expected ErrRoleRequestExists, got %v", err)
	}
}

func TestProfileReturnsCurrentApprovedRoles(t *testing.T) {
	user := &model.User{ID: uuid.New(), Email: "budi@example.com"}
	svc := service.NewAuthService(&fakeRepository{user: user, roles: []string{"borrower", "lender"}}, "test-secret")
	profile, err := svc.Profile(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if profile.User != user || len(profile.Roles) != 2 {
		t.Fatalf("unexpected profile: %#v", profile)
	}
}

func TestLoginRejectsInactiveUser(t *testing.T) {
	password := strings.Repeat("x", 12)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := &model.User{ID: uuid.New(), Email: "budi@example.com", PasswordHash: string(hash), Status: "blocked"}
	svc := service.NewAuthService(&fakeRepository{user: user}, "test-secret")
	_, err = svc.Login(context.Background(), user.Email, password)
	if !errors.Is(err, service.ErrUserInactive) {
		t.Fatalf("expected ErrUserInactive, got %v", err)
	}
}

func TestLoginRejectsIncorrectPassword(t *testing.T) {
	correctPassword := strings.Repeat("x", 12)
	hash, err := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := &model.User{ID: uuid.New(), Email: "budi@example.com", PasswordHash: string(hash), Status: "active"}
	svc := service.NewAuthService(&fakeRepository{user: user}, "test-secret")
	_, err = svc.Login(context.Background(), user.Email, strings.Repeat("y", 12))
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestRegisterRejectsExistingEmail(t *testing.T) {
	repo := &fakeRepository{user: &model.User{Email: "budi@example.com"}}
	svc := service.NewAuthService(repo, "test-secret")
	_, err := svc.Register(context.Background(), service.RegisterInput{FullName: "Budi", Email: "budi@example.com", Phone: "0812", Password: strings.Repeat("x", 12), City: "Jakarta", Address: "Alamat", TermsAccepted: true})
	if !errors.Is(err, service.ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}
