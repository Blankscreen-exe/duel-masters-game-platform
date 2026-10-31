package store

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// user0016 is the users table's avatar column as migration 0016 adds it. See
// package users.
type user0016 struct {
	// AvatarID is the picture the player chose, empty until they choose one.
	AvatarID string `gorm:"not null;default:''"`
}

func (user0016) TableName() string { return "users" }

// migration0016UserAvatars lets players choose a picture for their profile.
func migration0016UserAvatars() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0016_add_user_avatars",
		Migrate: func(tx *gorm.DB) error {
			return addColumns(tx, &user0016{}, "AvatarID")
		},
		Rollback: func(tx *gorm.DB) error {
			return dropColumns(tx, &user0016{}, map[string]any{"avatar_id": ""}, "AvatarID")
		},
	}
}
