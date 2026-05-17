package bot

import (
	"fmt"
	"log"

	"github.com/N1k3YB/GiftScheduleBot/db"
	"github.com/robfig/cron/v3"
	tele "gopkg.in/telebot.v3"
)

func StartScheduler() {
	c := cron.New()

	c.AddFunc("@hourly", func() {
		runNotifications("notify_3days", "⏰ Напоминание: розыгрыш завершается через 3 дня!")
		runNotifications("notify_1day", "⏰ Напоминание: розыгрыш завершается завтра!")
		runNotifications("notify_1hour", "⏰ Напоминание: розыгрыш завершается через час!")
		runExpiredCheck()
	})

	c.Start()
	log.Println("scheduler started")
}

func runNotifications(notifyField, header string) {
	posts, err := db.GetActivePendingNotification(notifyField)
	if err != nil {
		log.Printf("scheduler: get %s posts: %v", notifyField, err)
		return
	}
	for _, p := range posts {
		users, err := db.GetUsersForPost(p.ID, notifyField)
		if err != nil {
			continue
		}
		text := buildNotifyText(p, header)
		for _, u := range users {
			sendNotify(u.TelegramID, text)
		}
	}
}

func runExpiredCheck() {
	posts, err := db.GetExpiredPosts()
	if err != nil {
		log.Printf("scheduler: get expired: %v", err)
		return
	}
	for _, p := range posts {
		if err := db.MarkCompleted(p.ID); err != nil {
			log.Printf("scheduler: mark completed #%d: %v", p.ID, err)
			continue
		}
		users, err := db.GetUsersForPost(p.ID, "all")
		if err != nil {
			continue
		}
		text := buildCompletedText(p)
		for _, u := range users {
			sendNotify(u.TelegramID, text)
		}
	}
}

func buildNotifyText(p *db.Post, header string) string {
	title := p.Title
	if title == "" {
		title = "Без названия"
	}
	text := fmt.Sprintf("%s\n\n🎁 <b>%s</b>", header, escapeHTML(title))
	if p.EndDate != nil {
		text += fmt.Sprintf("\n📅 %s", p.EndDate.Format("02.01.2006 15:04"))
	}
	link := PostLink(p)
	if link != "" {
		text += fmt.Sprintf("\n🔗 <a href=\"%s\">Открыть пост</a>", link)
	}
	return text
}

func buildCompletedText(p *db.Post) string {
	title := p.Title
	if title == "" {
		title = "Без названия"
	}
	text := fmt.Sprintf("🏁 Розыгрыш завершён!\n\n🎁 <b>%s</b>", escapeHTML(title))

	resultsLink := ""
	if p.ResultsURL != "" && !p.ResultsInSamePost {
		resultsLink = p.ResultsURL
	} else {
		resultsLink = PostLink(p)
	}
	if resultsLink != "" {
		text += fmt.Sprintf("\n\n📋 <a href=\"%s\">Посмотреть итоги</a>", resultsLink)
	}
	text += "\n\n🔍 Открой бота и проверь результаты командой /start"
	return text
}

func sendNotify(telegramID int64, text string) {
	if B == nil {
		return
	}
	_, err := B.Send(
		&tele.Chat{ID: telegramID},
		text,
		&tele.SendOptions{ParseMode: tele.ModeHTML},
	)
	if err != nil {
		log.Printf("scheduler: send to %d: %v", telegramID, err)
	}
}
