package store

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// chatMessage0006 is the lobby_chat_messages table as migration 0006 creates
// it. See package chat.
type chatMessage0006 struct {
	ID     string   `gorm:"primaryKey;size:36"`
	UserID string   `gorm:"size:36;not null;index:idx_lobby_chat_messages_user_id"`
	User   user0002 `gorm:"constraint:OnDelete:CASCADE"`
	Text   string   `gorm:"type:text;not null"`
	// PostedAt orders the chat; the newest messages are the ones shown.
	PostedAt time.Time `gorm:"not null;index:idx_lobby_chat_messages_posted_at"`
	// DeletedAt is set when an admin removes the message, which then is no
	// longer shown.
	DeletedAt *time.Time
}

func (chatMessage0006) TableName() string { return "lobby_chat_messages" }

// migration0006LobbyChat creates the lobby chat's messages. A deleted account
// takes its messages with it.
func migration0006LobbyChat() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "0006_create_lobby_chat",
		Migrate: func(tx *gorm.DB) error {
			return tx.Migrator().CreateTable(&chatMessage0006{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&chatMessage0006{})
		},
	}
}
