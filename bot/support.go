package bot

import (
	"fmt"
	"sync"

	"github.com/N1k3YB/GiftScheduleBot/config"
	tele "gopkg.in/telebot.v3"
)

var (
	supportMu    sync.Mutex
	supportUsers = make(map[int64]bool)
	userToMsgID  = make(map[int64]int)
)

func supportStart(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return nil
	}
	supportMu.Lock()
	supportUsers[sender.ID] = true
	supportMu.Unlock()

	menu := B.NewMarkup()
	menu.Inline(menu.Row(menu.Data("✅ Проблема решена", "support_done")))
	return c.Send(
		"🆘 <b>Техподдержка</b>\n\nОпиши свою проблему — сообщения будут переданы в поддержку.\n\nКогда вопрос решён — нажми кнопку ниже.",
		&tele.SendOptions{ParseMode: tele.ModeHTML},
		menu,
	)
}

func supportDone(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return nil
	}

	name := sender.FirstName
	if sender.Username != "" {
		name = fmt.Sprintf("%s (@%s)", sender.FirstName, sender.Username)
	}

	supportMu.Lock()
	delete(supportUsers, sender.ID)
	delete(userToMsgID, sender.ID)
	supportMu.Unlock()

	if config.C.AdminChatID != 0 {
		B.Send(&tele.Chat{ID: config.C.AdminChatID},
			fmt.Sprintf("✅ Пользователь %s [<code>%d</code>] закрыл тикет — проблема решена.", escapeHTML(name), sender.ID),
			&tele.SendOptions{ParseMode: tele.ModeHTML},
		)
	}

	return c.Edit(
		"✅ Рады помочь! Обращайся, если что-то ещё нужно.",
		&tele.SendOptions{ParseMode: tele.ModeHTML},
	)
}

func isInSupport(userID int64) bool {
	supportMu.Lock()
	defer supportMu.Unlock()
	return supportUsers[userID]
}

func handleSupportMessage(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return nil
	}
	cfg := config.C
	if cfg.AdminChatID == 0 {
		return c.Send("⚠️ Техподдержка временно недоступна.")
	}

	name := sender.FirstName
	if sender.Username != "" {
		name = fmt.Sprintf("%s (@%s)", sender.FirstName, sender.Username)
	}

	fwdText := fmt.Sprintf(
		"📩 <b>Сообщение от пользователя</b> %s [<code>%d</code>]:\n\n%s",
		escapeHTML(name), sender.ID, escapeHTML(c.Text()),
	)

	adminChat := &tele.Chat{ID: cfg.AdminChatID}
	sent, err := B.Send(adminChat, fwdText, &tele.SendOptions{
		ParseMode:             tele.ModeHTML,
		DisableWebPagePreview: true,
	})
	if err == nil && sent != nil {
		supportMu.Lock()
		userToMsgID[sender.ID] = sent.ID
		supportMu.Unlock()
	}

	return c.Send("✉️ Сообщение передано в поддержку. Ожидай ответа.")
}

func handleAdminReply(c tele.Context) error {
	msg := c.Message()
	if msg == nil || msg.ReplyTo == nil {
		return nil
	}
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return nil
	}
	if msg.Chat == nil || msg.Chat.ID != config.C.AdminChatID {
		return nil
	}

	replyToID := msg.ReplyTo.ID

	supportMu.Lock()
	var targetUserID int64
	for uid, mid := range userToMsgID {
		if mid == replyToID {
			targetUserID = uid
			break
		}
	}
	supportMu.Unlock()

	if targetUserID == 0 {
		return nil
	}

	replyText := fmt.Sprintf("💬 <b>Ответ поддержки:</b>\n\n%s", escapeHTML(msg.Text))
	menu := B.NewMarkup()
	menu.Inline(menu.Row(menu.Data("✅ Проблема решена", "support_done")))
	B.Send(&tele.User{ID: targetUserID}, replyText, &tele.SendOptions{
		ParseMode: tele.ModeHTML,
	}, menu)
	return nil
}
