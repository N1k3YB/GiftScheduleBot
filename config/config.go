package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken    string
	DBPath      string
	DumpChatID  int64
	AdminChatID int64
	AdminIDs    []int64
}

var C Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	C.BotToken = mustGetenv("BOT_TOKEN")
	C.DBPath = getenvOr("DB_PATH", "./giftbot.db")

	if raw := os.Getenv("DUMP_CHAT_ID"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			C.DumpChatID = id
		}
	}

	if raw := os.Getenv("ADMIN_CHAT_ID"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			C.AdminChatID = id
		}
	}

	if raw := os.Getenv("ADMIN_IDS"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if id, err := strconv.ParseInt(part, 10, 64); err == nil {
				C.AdminIDs = append(C.AdminIDs, id)
			}
		}
	}
}

func IsAdmin(telegramID int64) bool {
	for _, id := range C.AdminIDs {
		if id == telegramID {
			return true
		}
	}
	return false
}

func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env variable %s is not set", key)
	}
	return v
}

func getenvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
