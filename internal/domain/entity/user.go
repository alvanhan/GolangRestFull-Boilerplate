package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type UserRole string
type UserStatus string

const (
	RoleSuperAdmin UserRole = "super_admin"
	RoleAdmin      UserRole = "admin"
	RoleManager    UserRole = "manager"
	RoleEditor     UserRole = "editor"
	RoleViewer     UserRole = "viewer"
)

const (
	StatusActive   UserStatus = "active"
	StatusInactive UserStatus = "inactive"
	StatusBanned   UserStatus = "banned"
)

// JSONMap is a custom type for JSONB columns that implements sql.Scanner and driver.Valuer.
type JSONMap map[string]interface{}

func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("JSONMap: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, j)
}

type User struct {
	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email            string     `gorm:"uniqueIndex;not null"`
	Username         string     `gorm:"uniqueIndex;not null"`
	FullName         string     `gorm:"not null"`
	PasswordHash     string     `gorm:"not null"`
	Role             UserRole   `gorm:"type:varchar(20);not null;default:'viewer'"`
	Status           UserStatus `gorm:"type:varchar(20);not null;default:'active'"`
	Avatar           *string
	StorageQuota     int64      `gorm:"default:10737418240"` // 10GB default
	StorageUsed      int64      `gorm:"default:0"`
	LastLoginAt      *time.Time
	LastLoginIP      *string
	EmailVerified    bool       `gorm:"default:false"`
	TwoFactorEnabled bool       `gorm:"default:false"`
	Metadata         JSONMap    `gorm:"type:jsonb"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time `gorm:"index"`
}

func (u *User) IsActive() bool      { return u.Status == StatusActive }
func (u *User) IsSuperAdmin() bool  { return u.Role == RoleSuperAdmin }
func (u *User) IsAdmin() bool       { return u.Role == RoleAdmin || u.Role == RoleSuperAdmin }
func (u *User) HasStorageSpace(size int64) bool {
	return u.StorageUsed+size <= u.StorageQuota
}
func (u *User) RoleLevel() int {
	switch u.Role {
	case RoleSuperAdmin:
		return 5
	case RoleAdmin:
		return 4
	case RoleManager:
		return 3
	case RoleEditor:
		return 2
	case RoleViewer:
		return 1
	default:
		return 0
	}
}

type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"not null;uniqueIndex"`
	UserAgent string
	IPAddress string
	ExpiresAt time.Time `gorm:"not null"`
	Revoked   bool      `gorm:"default:false"`
	CreatedAt time.Time

	User User `gorm:"foreignKey:UserID"`
}
