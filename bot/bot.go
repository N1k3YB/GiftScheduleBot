package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/N1k3YB/GiftScheduleBot/config"
	"github.com/N1k3YB/GiftScheduleBot/db"
	tele "gopkg.in/telebot.v3"
)

var B *tele.Bot

func Start() error {
	pref := tele.Settings{
		Token:  config.C.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	var err error
	B, err = tele.NewBot(pref)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	registerHandlers()
	B.Start()
	return nil
}

func registerHandlers() {
	B.Handle("/start", withAuth(handleStart))
	B.Handle("/admin", handleAdmin)
	B.Handle("/mylist", withAuth(handleMyList))

	B.Handle(tele.OnText, withAuth(handleText))
	B.Handle(tele.OnPhoto, withAuth(handleMedia))
	B.Handle(tele.OnVideo, withAuth(handleMedia))
	B.Handle(tele.OnDocument, withAuth(handleMedia))

	B.Handle(tele.OnQuery, handleInlineQuery)

	B.Handle(tele.OnCallback, handleCallback)
}

func withAuth(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		sender := c.Sender()
		if sender == nil {
			return nil
		}
		user, err := db.UpsertUser(sender.ID, sender.Username, sender.FirstName, sender.LastName)
		if err != nil {
			return nil
		}
		if user.IsBanned {
			return nil
		}
		c.Set("dbuser", user)
		return next(c)
	}
}

func getDBUser(c tele.Context) *db.User {
	if u, ok := c.Get("dbuser").(*db.User); ok {
		return u
	}
	return nil
}

func mainMenuMarkup() *tele.ReplyMarkup {
	menu := B.NewMarkup()
	menu.Inline(
		menu.Row(
			menu.Data("🎁 Добавить розыгрыш", "noop", "noop"),
		),
		menu.Row(
			menu.Data("📋 Мои розыгрыши", "my_list:0"),
			menu.Data("🌐 Все розыгрыши", "all_list:0"),
		),
		menu.Row(
			menu.Data("👤 Профиль", "profile"),
		),
		menu.Row(
			menu.Data("❌ Закрыть", "close"),
		),
	)
	return menu
}

func handleStart(c tele.Context) error {
	u := getDBUser(c)
	if u == nil {
		return nil
	}

	deep := c.Message().Payload
	if deep == "mylist" {
		return showMyList(c, u, 0)
	}

	text := fmt.Sprintf(
		"👋 Привет, <b>%s</b>!\n\n"+
			"Я помогаю отслеживать розыгрыши.\n\n"+
			"📌 <b>Как добавить розыгрыш:</b>\n"+
			"• Перешли пост с розыгрышем сюда\n"+
			"• Или отправь ссылку вида <code>t.me/channel/123</code>\n\n"+
			"📂 <b>Общий список</b> формируется из розыгрышей, которые сохранили пользователи бота.\n\n"+
			"Используй кнопки ниже 👇",
		u.FirstName,
	)
	return c.Send(text, &tele.SendOptions{ParseMode: tele.ModeHTML}, mainMenuMarkup())
}

func PostLink(p *db.Post) string {
	if p.ChannelUsername != "" {
		return fmt.Sprintf("https://t.me/%s/%d", strings.TrimPrefix(p.ChannelUsername, "@"), p.TelegramMsgID)
	}
	if p.SourceURL != "" {
		return p.SourceURL
	}
	return ""
}

func PostShortText(p *db.Post) string {
	title := p.Title
	if title == "" {
		title = "Без названия"
	}
	link := PostLink(p)
	endStr := ""
	if p.EndDate != nil {
		endStr = fmt.Sprintf(" · до %s", p.EndDate.Format("02.01.2006"))
	}
	status := ""
	if p.IsCompleted {
		status = " ✅"
	}
	if link != "" {
		return fmt.Sprintf("<a href=\"%s\">%s</a>%s%s", link, title, endStr, status)
	}
	return fmt.Sprintf("%s%s%s", title, endStr, status)
}
