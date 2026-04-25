package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        uuid.UUID  `json:"id" db:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()" rbac:"read:*;write:*"`
	Firstname string     `json:"firstname" db:"firstname" gorm:"not null" rbac:"read:*;write:any"`
	Lastname  string     `json:"lastname" db:"lastname" gorm:"not null" rbac:"read:*;write:any"`
	Email     string     `json:"email" db:"email" gorm:"uniqueIndex;not null" rbac:"read:*;write:any"`
	Password  *string    `json:"-" db:"password" rbac:"read:none;write:any"`
	Role      string     `json:"role" db:"role" gorm:"not null;default:'user'" rbac:"read:*;write:admin"`
	CreatedAt time.Time  `json:"created_at" db:"created_at" rbac:"read:*;write:none"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" db:"updated_at" rbac:"read:*;write:none"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) HashPassword() error {
	if u.Password == nil || *u.Password == "" {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	hashedStr := string(hashedPassword)
	u.Password = &hashedStr
	return nil
}

func (u *User) CheckPassword(password string) bool {
	if u.Password == nil {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(*u.Password), []byte(password))
	return err == nil
}
