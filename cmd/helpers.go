package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	UP   = "UP"
	DOWN = "DOWN"
)

func (b *bot) Poller(url string) {

	tick := time.NewTicker(30 * time.Second)

	defer tick.Stop()

	state := UP

	site := strings.TrimRight(b.config.URL, "v1/ping")

	for range tick.C {

		client := &http.Client{
			Timeout: 9 * time.Second,
		}

		resp, err := client.Get(url)

		if err != nil {
			slog.Error("Poller", "error perform get request", err.Error())
			continue
		}

		if resp.StatusCode != 200 {
			if state == UP {
				err := b.sendMsgToTelegramIds(fmt.Sprintf("%s is down 😞 (bring me up please).", site))

				if err != nil {
					slog.Error("Poller", "error sending msg", err.Error())
				}

				state = DOWN
			}
			continue
		}

		if state == DOWN {
			err = b.sendMsgToTelegramIds(fmt.Sprintf("%s is up and running 🔥", site))
			if err != nil {
				slog.Error("Poller", "error sending msg", err.Error())
			}

			state = UP

		}

	}
}

func (b *bot) sendMsgToTelegramIds(msg string) error {

	ids, err := b.models.Users.GetActiveUsers()

	if err != nil {
		slog.Error("Failed to get active users")
		return err
	}

	for _, id := range ids {
		msg := tgbotapi.NewMessage(id.Id, msg)

		b.bot_api.Send(msg)
	}

	return nil
}
