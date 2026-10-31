package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// user0017 is the users table's email columns as migration 0017 adds them. See
// package users.
type user0017 struct {
	// Email is the address as the player typed it; empty for computer
	// opponents and accounts older than this migration.
	Email string `gorm:"not null;default:''"`
	// EmailKey is the lowercase form that keeps addresses unique, among those
	// that exist.
	EmailKey string `gorm:"not null;default:'';uniqueIndex:users_email_key_unique,where:email_key <> ''"`
	// EmailVerifiedAt is when the address was confirmed; nil keeps the player
	// out until it is. Accounts older than this migration count as confirmed.
	EmailVerifiedAt *time.Time
	// VerifyBy is when an account still not confirmed is deleted; nil for one
	// that has never been sent a code.
	VerifyBy *time.Time
}

func (user0017) TableName() string { return "users" }

// emailVerification0017 is the code a player has been sent. See package auth.
type emailVerification0017 struct {
	UserID    string    `gorm:"primaryKey"`
	User      user0002  `gorm:"constraint:OnDelete:CASCADE"`
	CodeHash  string    `gorm:"not null"`
	SentAt    time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null"`
	Attempts  int       `gorm:"not null;default:0"`
}

func (emailVerification0017) TableName() string { return "email_verifications" }

// mailSettings0017 is the mail server's settings, sealed together with
// APP_SECRET_KEY. See package mail.
type mailSettings0017 struct {
	ID        int       `gorm:"primaryKey;autoIncrement:false"`
	Sealed    string    `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (mailSettings0017) TableName() string { return "mail_settings" }

// migration0017Email gives players an email address to confirm with a code, and
// the site a mail server to send it through.
func migration0017Email() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0017_add_email",
		Migrate: func(tx *gorm.DB) error {
			if err := addColumns(tx, &user0017{}, "Email", "EmailKey", "EmailVerifiedAt", "VerifyBy"); err != nil {
				return err
			}
			if err := tx.Migrator().CreateIndex(&user0017{}, "users_email_key_unique"); err != nil {
				return err
			}

			// Every account so far stays usable: none has an address to confirm.
			err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).
				Model(&user0017{}).
				Where("email_verified_at IS NULL").
				Update("email_verified_at", tx.NowFunc()).Error
			if err != nil {
				return err
			}

			return tx.Migrator().CreateTable(&emailVerification0017{}, &mailSettings0017{})
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Migrator().DropTable(&emailVerification0017{}, &mailSettings0017{}); err != nil {
				return err
			}
			if err := tx.Migrator().DropIndex(&user0017{}, "users_email_key_unique"); err != nil {
				return err
			}

			starting := map[string]any{"email": "", "email_key": "", "email_verified_at": nil, "verify_by": nil}
			return dropColumns(tx, &user0017{}, starting, "Email", "EmailKey", "EmailVerifiedAt", "VerifyBy")
		},
	}
}
