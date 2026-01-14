package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const SleepTimeout = 5 * time.Second

type TGBot struct {
	N8NWebhook   string
	Token        string
	AllowedUsers []string
	client       *http.Client
}

func NewBot(token, n8nWebhook string, allowedUsers []string) *TGBot {
	return &TGBot{
		Token:        token,
		N8NWebhook:   n8nWebhook,
		AllowedUsers: allowedUsers,
		client: &http.Client{
			Timeout: time.Second * 5,
		},
	}
}

func (tg *TGBot) Start(ctx context.Context) {
	slog.Info("Starting telegram bot")
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	b, err := bot.New(tg.Token, bot.WithDefaultHandler(tg.handler))
	if err != nil {
		slog.Error("Failed to create telegram bot", "error", err)
		os.Exit(1)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/id", bot.MatchTypeExact, tg.idHandler)
	b.Start(ctx)
}

func (tg *TGBot) handler(ctx context.Context, _ *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	from := update.Message.From.ID

	if !slices.Contains(tg.AllowedUsers, strconv.FormatInt(from, 10)) {
		slog.Info("User not allowed", "user", from)
		return
	}

	body, err := json.Marshal(update)
	if err != nil {
		slog.Error("Failed to serialize update", "error", err)
		return
	}

	go func() {
		for i := 0; i < 3; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				// Continue to next retry
			}

			err := tg.notifyN8n(ctx, body)
			if err != nil {
				slog.Error("Failed to notify n8n, retrying", "error", err, "attempt", i+1)
				select {
				case <-ctx.Done():
					return
				case <-time.After(SleepTimeout):
					// Continue to next retry
				}
				continue
			}

			return
		}
	}()
}

func (tg *TGBot) notifyN8n(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", tg.N8NWebhook, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := tg.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send http request: %w", err)
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	return fmt.Errorf("failed to send http request: %s", resp.Status)
}

func (tg *TGBot) idHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   strconv.FormatInt(update.Message.From.ID, 10),
	})
}
