package repository

import (
	"gorm.io/gorm"
	"shopping-cart/infrastructure"
	"shopping-cart/model/database"
	"time"
)

type UserRepository interface {
	Create(user *database.User) error
	FindByID(id int) (*database.User, error)
	FindAll() ([]database.User, error)
	Update(user *database.User) error
	Delete(user *database.User) error
	FindByLineID(lineID string) (*database.User, error)
	DeleteTx(tx *gorm.DB, id int) error
	SoftDeleteTx(tx *gorm.DB, id int) error
	FindByIDIncludingDeleted(id int) (*database.User, error) // 新增的方法
	BeginTransaction() *gorm.DB
	ListUsers() ([]database.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository() UserRepository {
	return &userRepository{
		db: infrastructure.Db,
	}
}

func (r *userRepository) Create(user *database.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) FindByID(id int) (*database.User, error) {
	var user database.User
	err := r.db.Where("id = ? AND is_deleted = ?", id, false).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByIDIncludingDeleted(id int) (*database.User, error) {
	var user database.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindAll() ([]database.User, error) {
	var users []database.User
	err := r.db.Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepository) Update(user *database.User) error {
	return r.db.Updates(user).Error
}

func (r *userRepository) Delete(user *database.User) error {
	return r.db.Delete(user).Error
}

func (r *userRepository) FindByLineID(lineID string) (*database.User, error) {
	var user database.User
	err := r.db.Where("line_id = ? AND is_deleted = ?", lineID, false).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) DeleteTx(tx *gorm.DB, id int) error {
	var user database.User
	err := tx.First(&user, id).Error
	if err != nil {
		return err
	}
	return tx.Delete(&user).Error
}

func (r *userRepository) SoftDeleteTx(tx *gorm.DB, id int) error {
	var user database.User
	err := tx.First(&user, id).Error
	if err != nil {
		return err
	}
	now := time.Now()
	user.DeletedAt = &now
	user.IsDeleted = true
	return tx.Save(&user).Error
}

func (r *userRepository) BeginTransaction() *gorm.DB {
	return r.db.Begin()
}

func (r *userRepository) ListUsers() ([]database.User, error) {
	var users []database.User
	err := r.db.Where("is_deleted = ?", false).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
