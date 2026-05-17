package bot

import (
	"fmt"
	"strings"

	"github.com/N1k3YB/GiftScheduleBot/db"
	tele "gopkg.in/telebot.v3"
)

func handleInlineQuery(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Answer(&tele.QueryResponse{})
	}

	u, err := db.GetUserByTelegramID(sender.ID)
	if err != nil || u == nil || u.IsBanned {
		return c.Answer(&tele.QueryResponse{})
	}

	posts, err := db.GetLastUserPosts(u.ID, 6)
	if err != nil || len(posts) == 0 {
		results := tele.Results{
			&tele.ArticleResult{
				Title:       "У тебя нет сохранённых розыгрышей",
				Description: "Добавь розыгрыши в боте",
				Text:        "Пока нет сохранённых розыгрышей. Открой бота и добавь первый!",
			},
		}
		results[0].SetResultID("empty")
		return c.Answer(&tele.QueryResponse{Results: results, CacheTime: 1})
	}

	showAll := len(posts) > 5
	display := posts
	if showAll {
		display = posts[:5]
	}

	uname := u.Username
	if uname == "" {
		uname = fmt.Sprintf("%s %s", u.FirstName, u.LastName)
	} else {
		uname = "@" + uname
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎁 <b>Розыгрыши %s</b>\n\n", uname))
	for i, up := range display {
		p := &up.Post
		title := p.Title
		if title == "" {
			title = "Без названия"
		}
		link := PostLink(p)
		endStr := ""
		if p.EndDate != nil {
			endStr = fmt.Sprintf(" · до %s", p.EndDate.Format("02.01"))
		}
		if link != "" {
			sb.WriteString(fmt.Sprintf("%d. <a href=\"%s\">%s</a>%s\n", i+1, link, escapeHTML(title), endStr))
		} else {
			sb.WriteString(fmt.Sprintf("%d. %s%s\n", i+1, escapeHTML(title), endStr))
		}
	}

	var markup *tele.ReplyMarkup
	if showAll {
		markup = B.NewMarkup()
		deepLink := fmt.Sprintf("https://t.me/%s?start=mylist", B.Me.Username)
		markup.Inline(markup.Row(markup.URL("👀 Смотреть все", deepLink)))
	}

	article := &tele.ArticleResult{
		Title:       fmt.Sprintf("Отправить мои розыгрыши (%d)", len(posts)),
		Description: "Поделиться списком сохранённых розыгрышей",
		Text:        sb.String(),
	}
	article.SetResultID(fmt.Sprintf("mylist_%d", u.ID))
	if markup != nil {
		article.ReplyMarkup = markup
	}
	article.ParseMode = tele.ModeHTML

	return c.Answer(&tele.QueryResponse{
		Results:  tele.Results{article},
		CacheTime: 1,
	})
}
