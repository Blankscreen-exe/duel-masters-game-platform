package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// user0002 is the users table as migration 0002 creates it.
type user0002 struct {
	ID string `gorm:"primaryKey;size:36"`
	// Username is the name as the player typed it; UsernameKey is its
	// lowercase form, which keeps names unique regardless of case.
	Username     string    `gorm:"size:20;not null"`
	UsernameKey  string    `gorm:"size:20;not null;uniqueIndex:idx_users_username_key"`
	PasswordHash string    `gorm:"type:text;not null"`
	Role         string    `gorm:"size:10;not null;check:chk_users_role,role IN ('player','admin')"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

func (user0002) TableName() string { return "users" }

// session0002 is the sessions table as migration 0002 creates it. ID is the
// SHA-256 of the session token, never the token itself.
type session0002 struct {
	ID        string    `gorm:"primaryKey;size:64"`
	UserID    string    `gorm:"size:36;not null;index:idx_sessions_user_id"`
	User      user0002  `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null;index:idx_sessions_expires_at"`
}

func (session0002) TableName() string { return "sessions" }

// migration0002Accounts creates the player accounts and their login
// sessions. A deleted account takes its sessions with it.
func migration0002Accounts() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0002_create_accounts",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().CreateTable(&user0002{}, &session0002{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&session0002{}, &user0002{})
		},
	}
}
