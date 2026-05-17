package main

import (
	"log"

	"github.com/nkeydash/GiftScheduleBot/bot"
	"github.com/nkeydash/GiftScheduleBot/config"
	"github.com/nkeydash/GiftScheduleBot/db"
)

func main() {
	config.Load()
	db.Open(config.C.DBPath)
	bot.StartScheduler()
	log.Println("starting bot...")
	if err := bot.Start(); err != nil {
		log.Fatalf("bot error: %v", err)
	}
}
