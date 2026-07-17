package repository

import (
	"context"

	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	CreateUser(context.Context, *model.User) error
	FindUserByEmail(context.Context, string) (*model.User, error)
	FindUserByID(context.Context, uuid.UUID) (*model.User, error)
	FindRoleByName(context.Context, string) (*model.Role, error)
	CreateRoleRequest(context.Context, *model.RoleRequest) error
	HasOpenRoleRequest(context.Context, uuid.UUID, int) (bool, error)
	GetApprovedRoles(context.Context, uuid.UUID) ([]string, error)
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
