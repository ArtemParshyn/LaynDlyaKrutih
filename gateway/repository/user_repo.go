package repository

import (
	"errors"
	"gateway/entity"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var (
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	ErrUserNotFound      = errors.New("user not found")
)

type UserRepositoryInterface interface {
	Create(user *entity.User) error
	GetByID(id uint) (*entity.User, error)
	GetByEmail(email string) (*entity.User, error)
	GetAll() ([]*entity.User, error)
	Update(user *entity.User) error
	Delete(id uint) error
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *entity.User) error {
	tx := r.db.Create(user)
	if tx.Error != nil {
		if isDuplicate(tx.Error) {
			return ErrUserAlreadyExists
		}
		return tx.Error
	}
	return nil
}

func isDuplicate(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "sqlstate 23505")
}

func (r *UserRepository) GetByID(id uint) (*entity.User, error) {
	var user entity.User
	if err := r.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(email string) (*entity.User, error) {
	var user entity.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetAll() ([]*entity.User, error) {
	var users []*entity.User
	if err := r.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) Update(user *entity.User) error {
	tx := r.db.Save(user)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (r *UserRepository) Delete(id uint) error {
	tx := r.db.Delete(&entity.User{}, id)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}
