package domain

import "time"

type ID string

type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelTelegram Channel = "telegram"
)

type NotifySettings struct {
	EmailAlerts    bool
	TelegramChatID string
}

type User struct {
	ID           ID
	Name         string
	Email        string
	PasswordHash string
	GoogleSub    string
	StatusSlug   string
	Notify       NotifySettings
	CreatedAt    time.Time
}
