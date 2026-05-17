package main

import (
	"log"

	"github.com/N1k3YB/GiftScheduleBot/bot"
	"github.com/N1k3YB/GiftScheduleBot/config"
	"github.com/N1k3YB/GiftScheduleBot/db"
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
