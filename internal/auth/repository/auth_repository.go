package repository

import (
	"context"
	"time"

	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	CreateUser(context.Context, *model.User) error
	FindUserByEmail(context.Context, string) (*model.User, error)
	FindUserByID(context.Context, uuid.UUID) (*model.User, error)
	FindUserByGoogleID(context.Context, string) (*model.User, error)
	FindRoleByName(context.Context, string) (*model.Role, error)
	CreateRoleRequest(context.Context, *model.RoleRequest) error
	HasOpenRoleRequest(context.Context, uuid.UUID, int) (bool, error)
	GetApprovedRoles(context.Context, uuid.UUID) ([]string, error)
	GetRoleRequests(ctx context.Context, status string) ([]model.RoleRequest, error)
	FindRoleRequestByID(ctx context.Context, id uuid.UUID) (*model.RoleRequest, error)
	UpdateRoleRequest(ctx context.Context, request *model.RoleRequest) error
	ApproveUserRole(ctx context.Context, userID uuid.UUID, roleID int, adminID uuid.UUID) error
}

type AuthRepository struct{ db *gorm.DB }

func NewAuthRepository(db *gorm.DB) *AuthRepository { return &AuthRepository{db: db} }

func (r *AuthRepository) CreateUser(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *AuthRepository) FindUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindUserByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("google_id = ?", googleID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindRoleByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *AuthRepository) CreateRoleRequest(ctx context.Context, request *model.RoleRequest) error {
	return r.db.WithContext(ctx).Create(request).Error
}

func (r *AuthRepository) HasOpenRoleRequest(ctx context.Context, userID uuid.UUID, roleID int) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.RoleRequest{}).
		Where("user_id = ? AND role_id = ? AND status IN ?", userID, roleID, []string{"submitted", "under_review", "revision_required"}).
		Count(&count).Error
	return count > 0, err
}

func (r *AuthRepository) GetApprovedRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var roles []string
	err := r.db.WithContext(ctx).Model(&model.UserRole{}).
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND user_roles.status = ?", userID, "approved").
		Pluck("roles.name", &roles).Error
	return roles, err
}

func (r *AuthRepository) GetRoleRequests(ctx context.Context, status string) ([]model.RoleRequest, error) {
	var requests []model.RoleRequest
	query := r.db.WithContext(ctx).Preload("User").Preload("Role")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

func (r *AuthRepository) FindRoleRequestByID(ctx context.Context, id uuid.UUID) (*model.RoleRequest, error) {
	var req model.RoleRequest
	if err := r.db.WithContext(ctx).Preload("User").Preload("Role").First(&req, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *AuthRepository) UpdateRoleRequest(ctx context.Context, request *model.RoleRequest) error {
	return r.db.WithContext(ctx).Save(request).Error
}

func (r *AuthRepository) ApproveUserRole(ctx context.Context, userID uuid.UUID, roleID int, adminID uuid.UUID) error {
	now := time.Now().UTC()
	var userRole model.UserRole
	err := r.db.WithContext(ctx).Where("user_id = ? AND role_id = ?", userID, roleID).First(&userRole).Error
	if err == gorm.ErrRecordNotFound {
		userRole = model.UserRole{
			UserID:     userID,
			RoleID:     roleID,
			Status:     "approved",
			ApprovedBy: &adminID,
			ApprovedAt: &now,
		}
		return r.db.WithContext(ctx).Create(&userRole).Error
	} else if err != nil {
		return err
	}

	userRole.Status = "approved"
	userRole.ApprovedBy = &adminID
	userRole.ApprovedAt = &now
	return r.db.WithContext(ctx).Save(&userRole).Error
}
