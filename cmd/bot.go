package main

import (
	"context"
	"database/sql"
	"flag"
	"github/Hopertz/is_up/postgres"
	"log"
	"log/slog"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

func init() {

	var programLevel = new(slog.LevelVar)

	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel})
	slog.SetDefault(slog.New(h))

}

type cfg struct {
	BOT_TOKEN string
	URL       string
	DSN       string
}

type bot struct {
	config  cfg
	bot_api *tgbotapi.BotAPI
	models  postgres.Models
}

const start_txt = "Use this bot to check if iim.sojirotech.com api service is up or down. Type /stop to stop receiving notifications`"
const stop_txt = "Sorry to see you leave You wont be receiving notifications. Type /start to receive"

const unknown_cmd = "i don't know that command"

func main() {

	var cfg cfg

	flag.StringVar(&cfg.BOT_TOKEN, "BOT-KEY", os.Getenv("BOT_TOKEN"), "BOT TOKEN")
	flag.StringVar(&cfg.URL, "URL", os.Getenv("URL"), "URL")
	flag.StringVar(&cfg.DSN, "db-dsn", os.Getenv("DSN_BOT"), "Postgres DSN")

	if cfg.BOT_TOKEN == "" || cfg.URL == "" || cfg.DSN == "" {
		log.Fatal("BOT_TOKEN, URL and DSN_BOT are required")
	}

	bot_api, err := tgbotapi.NewBotAPI(cfg.BOT_TOKEN)
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot_api.GetUpdatesChan(u)

	db, err := openDB(cfg.DSN)

	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	models := postgres.NewModels(db)

	bot := bot{
		config:  cfg,
		bot_api: bot_api,
		models:  models,
	}

	go bot.Poller(cfg.URL)

	for update := range updates {
		if update.Message == nil { // ignore non-Message updates
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		// Create a new MessageConfig. We don't have text yet,
		// so we leave it empty.
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		// Extract the command from the Message
		switch update.Message.Command() {
		case "start":
			msg.Text = start_txt
			botUser := &postgres.User{
				ID:       update.Message.From.ID,
				Isactive: true,
			}
			err := bot.models.Users.Insert(botUser)

			if err != nil {
				switch {
				case err.Error() == `pq: duplicate key value violates unique constraint "users_pkey"`:

					err := bot.models.Users.Update(botUser)
					if err != nil {
						slog.Error("err", "error updating user(insert)", err.Error())
					}

				default:
					slog.Error("err", "error inserting user", err.Error())
				}
			}

		case "stop":
			botUser := &postgres.User{
				ID:       update.Message.From.ID,
				Isactive: false,
			}
			err := bot.models.Users.Update(botUser)
			if err != nil {
				slog.Error("err", "error updating user(disable)", err.Error())
			}
			msg.Text = stop_txt

		case "help":
			msg.Text = `
			Commands for this @sojiro_bot bot are:
			
			/start  start the bot (i.e., enable receiving notifications)
			/stop   stop the bot (i.e., disable receiving notifications)
			/help   this help text
			`

		default:
			msg.Text = unknown_cmd
		}

		if _, err := bot.bot_api.Send(msg); err != nil {
			slog.Error("Bot", "error sending msg", err.Error())
		}


	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	db.PingContext(ctx)

	if err != nil {
		return nil, err
	}
	return db, err
}
