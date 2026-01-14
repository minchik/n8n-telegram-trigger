package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Token        string
	Webhook      string
	AllowedUsers []string
}

func config() Config {
	allowedUsersConf := strings.TrimSpace(os.Getenv("ALLOWED_USERS"))
	var allowedUsers []string
	if allowedUsersConf != "" {
		allowedUsers = strings.Split(allowedUsersConf, ",")
	} else {
		slog.Warn("No allowed users configured, no messages will be processed")
	}
	return Config{
		Token:        os.Getenv("TELEGRAM_TOKEN"),
		Webhook:      os.Getenv("N8N_WEBHOOK"),
		AllowedUsers: allowedUsers,
	}
}

func main() {
	c := config()
	tg := NewBot(c.Token, c.Webhook, c.AllowedUsers)
	tg.Start(context.Background())
}
