package bot

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/N1k3YB/GiftScheduleBot/config"
	"github.com/N1k3YB/GiftScheduleBot/db"
	"github.com/N1k3YB/GiftScheduleBot/parser"
	tele "gopkg.in/telebot.v3"
)

var reTMeLink = regexp.MustCompile(`(?i)https?://t\.me/([a-zA-Z0-9_]+)/(\d+)`)

func handleText(c tele.Context) error {
	msg := c.Message()
	if msg == nil {
		return nil
	}
	sender := c.Sender()
	if sender != nil {
		if msg.Chat != nil && msg.Chat.ID == config.C.AdminChatID && msg.ReplyTo != nil {
			return handleAdminReply(c)
		}
		if handled := HandleAdminText(c, sender.ID); handled {
			return nil
		}
		if isInSupport(sender.ID) {
			return handleSupportMessage(c)
		}
	}
	u := getDBUser(c)
	if u == nil {
		return nil
	}

	if msg.IsForwarded() {
		return processForward(c, u, msg)
	}

	text := parser.NormalizeText(msg.Text)
	if m := reTMeLink.FindStringSubmatch(text); m != nil {
		return processLink(c, u, m[1], parseInt64(m[2]), text)
	}

	if parser.IsGiveawayText(text) {
		return processGiveawayText(c, u, msg, text, 0, 0, "", "")
	}

	return c.Send("Привет! Пришли мне пост с розыгрышем (перешли или отправь ссылку t.me/channel/123), и я его сохраню 🎁", mainMenuMarkup())
}

func handleMedia(c tele.Context) error {
	msg := c.Message()
	if msg == nil {
		return nil
	}
	u := getDBUser(c)
	if u == nil {
		return nil
	}

	if msg.IsForwarded() {
		return processForward(c, u, msg)
	}

	caption := msg.Caption
	if caption == "" {
		return nil
	}
	if m := reTMeLink.FindStringSubmatch(caption); m != nil {
		return processLink(c, u, m[1], parseInt64(m[2]), caption)
	}
	if parser.IsGiveawayText(caption) {
		return processGiveawayText(c, u, msg, caption, 0, 0, "", "")
	}
	return nil
}

func processForward(c tele.Context, u *db.User, msg *tele.Message) error {
	text := parser.NormalizeText(msg.Text)
	if text == "" {
		text = parser.NormalizeText(msg.Caption)
	}

	var channelID int64
	var msgID int64
	var channelUsername string

	if msg.OriginalChat != nil {
		channelID = msg.OriginalChat.ID
		channelUsername = msg.OriginalChat.Username
		msgID = int64(msg.OriginalMessageID)
	}

	if !parser.IsGiveawayText(text) {
		return c.Send("⚠️ Этот пост не является розыгрышем")
	}

	return processGiveawayText(c, u, msg, text, channelID, msgID, channelUsername, "")
}

func processLink(c tele.Context, u *db.User, channelUser string, msgID int64, rawURL string) error {
	sourceURL := fmt.Sprintf("https://t.me/%s/%d", channelUser, msgID)

	var text string
	if config.C.DumpChatID != 0 {
		dumped, err := B.Forward(&tele.Chat{ID: config.C.DumpChatID}, &tele.Message{
			Chat: &tele.Chat{Username: channelUser},
			ID:   int(msgID),
		})
		if err == nil && dumped != nil {
			text = dumped.Text
			if text == "" {
				text = dumped.Caption
			}
			B.Delete(dumped)
		}
	}

	if text == "" {
		p := &db.Post{
			TelegramMsgID:   msgID,
			ChannelUsername: channelUser,
			SourceURL:       sourceURL,
			ContentParsed:   false,
			Title:           "Розыгрыш из " + channelUser,
		}
		created, isNew, err := db.CreateOrGetPost(p)
		if err != nil {
			return c.Send("❌ Ошибка при сохранении")
		}
		return linkAndReply(c, u, created, isNew, false)
	}

	return processGiveawayText(c, u, nil, text, 0, msgID, channelUser, sourceURL)
}

func processGiveawayText(c tele.Context, u *db.User, _ *tele.Message, text string, channelID, msgID int64, channelUser, sourceURL string) error {
	info := parser.ParseGiveaway(text)
	if !info.IsGiveaway {
		return c.Send("⚠️ Данный пост не является розыгрышем")
	}
	if info.EndDate != nil && info.EndDate.Before(time.Now()) {
		return c.Send("⚠️ Дата завершения этого розыгрыша уже прошла.")
	}

	p := &db.Post{
		TelegramMsgID:     msgID,
		ChannelUsername:   channelUser,
		ChannelID:         channelID,
		Title:             info.Title,
		EndDate:           info.EndDate,
		HasEndTime:        info.HasEndTime,
		ResultsInSamePost: info.ResultsInSamePost,
		SourceURL:         sourceURL,
		ContentParsed:     true,
		RawText:           text,
	}
	if len(info.Prizes) > 0 {
		p.Prizes = strings.Join(info.Prizes, "\n")
	}

	created, isNew, err := db.CreateOrGetPost(p)
	if err != nil {
		return c.Send("❌ Ошибка при сохранении.")
	}
	return linkAndReply(c, u, created, isNew, true)
}

func linkAndReply(c tele.Context, u *db.User, p *db.Post, isNewPost bool, parsed bool) error {
	alreadyHas, _ := db.UserAlreadyHasPost(u.ID, p.ID)
	if !alreadyHas {
		if err := db.AddUserPost(u.ID, p.ID); err != nil {
			return c.Send("❌ Ошибка при добавлении.")
		}
	}

	var sb strings.Builder
	if alreadyHas {
		sb.WriteString("ℹ️ Этот розыгрыш уже есть в списке.\n\n")
	} else if isNewPost {
		sb.WriteString("✅ Розыгрыш добавлен!\n\n")
	} else {
		sb.WriteString("✅ Розыгрыш добавлен!\n\n")
	}

	sb.WriteString(fmt.Sprintf("🆔 ID: <b>#%d</b>\n", p.ID))
	sb.WriteString(fmt.Sprintf("📌 <b>%s</b>\n", escapeHTML(p.Title)))

	if p.EndDate != nil {
		sb.WriteString(fmt.Sprintf("📅 Завершится: <b>%s</b>\n", p.EndDate.Format("02.01.2006 15:04")))
	}
	if p.Prizes != "" && parsed {
		prizes := strings.Split(p.Prizes, "\n")
		if len(prizes) > 0 {
			sb.WriteString("🎁 Призы:\n")
			for _, pr := range prizes {
				if pr != "" {
					sb.WriteString(fmt.Sprintf("  • %s\n", pr))
				}
			}
		}
	}
	link := PostLink(p)
	if link != "" {
		sb.WriteString(fmt.Sprintf("🔗 <a href=\"%s\">Открыть пост</a>\n", link))
	}
	if !parsed {
		sb.WriteString("\n⚠️ Содержимое поста недоступно. Сохранена только ссылка.")
	}

	menu := B.NewMarkup()
	menu.Inline(
		menu.Row(menu.Data("📋 Мои розыгрыши", "my_list:0")),
		menu.Row(menu.Data("🏠 Главное меню", "main_menu")),
	)
	return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML, DisableWebPagePreview: true}, menu)
}

func handleMyList(c tele.Context) error {
	u := getDBUser(c)
	if u == nil {
		return nil
	}
	return showMyList(c, u, 0)
}

func showMyList(c tele.Context, u *db.User, page int) error {
	const perPage = 5
	total, _ := db.CountUserPosts(u.ID)
	if total == 0 {
		menu := B.NewMarkup()
		menu.Inline(menu.Row(menu.Data("🏠 Главное меню", "main_menu")))
		return c.Send("📋 У тебя пока нет сохранённых розыгрышей.\n\nПришли пост или ссылку 🎁", menu)
	}

	totalPages := (total + perPage - 1) / perPage
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	posts, err := db.GetUserPosts(u.ID, perPage, page*perPage)
	if err != nil {
		return c.Send("❌ Ошибка загрузки.")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 <b>Мои розыгрыши</b> (стр. %d/%d)\n\n", page+1, totalPages))
	for i, up := range posts {
		p := &up.Post
		num := page*perPage + i + 1
		endStr := ""
		if p.EndDate != nil {
			endStr = fmt.Sprintf(" · до %s", p.EndDate.Format("02.01"))
		}
		status := ""
		if p.IsCompleted {
			status = " ✅"
		}
		link := PostLink(p)
		title := p.Title
		if title == "" {
			title = "Без названия"
		}
		if link != "" {
			sb.WriteString(fmt.Sprintf("%d. <a href=\"%s\">%s</a>%s%s\n", num, link, escapeHTML(title), endStr, status))
		} else {
			sb.WriteString(fmt.Sprintf("%d. %s%s%s\n", num, escapeHTML(title), endStr, status))
		}
	}

	menu := B.NewMarkup()
	var rows []tele.Row

	for _, up := range posts {
		p := &up.Post
		shortTitle := []rune(p.Title)
		if len(shortTitle) > 18 {
			shortTitle = append(shortTitle[:18], '…')
		}
		btnLabel := string(shortTitle)
		if btnLabel == "" {
			btnLabel = fmt.Sprintf("#%d", p.ID)
		}
		btnCheck := menu.Data(fmt.Sprintf("🔍 %s", btnLabel), fmt.Sprintf("check_result:%d", p.ID))
		btnDel := menu.Data("🗑", fmt.Sprintf("del_my:%d", up.UserPostID))
		rows = append(rows, menu.Row(btnCheck, btnDel))
	}

	navRow := buildNavRow(menu, "my_list", page, totalPages)
	rows = append(rows, navRow)
	rows = append(rows, menu.Row(menu.Data("🏠 Главное меню", "main_menu")))
	rows = append(rows, menu.Row(menu.Data("❌ Закрыть", "close")))
	menu.Inline(rows...)

	return editOrSend(c, sb.String(), menu)
}

func buildNavRow(menu *tele.ReplyMarkup, prefix string, page, total int) tele.Row {
	prev := menu.Data("⬅️", fmt.Sprintf("%s:%d", prefix, page-1))
	counter := menu.Data(fmt.Sprintf("%d/%d", page+1, total), "noop:page")
	next := menu.Data("➡️", fmt.Sprintf("%s:%d", prefix, page+1))
	return menu.Row(prev, counter, next)
}

func editOrSend(c tele.Context, text string, markup *tele.ReplyMarkup) error {
	opts := &tele.SendOptions{ParseMode: tele.ModeHTML, DisableWebPagePreview: true}
	if c.Callback() != nil {
		return c.Edit(text, opts, markup)
	}
	return c.Send(text, opts, markup)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func parseInt64(s string) int64 {
	var n int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		}
	}
	return n
}
