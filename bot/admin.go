package bot

import (
	"fmt"
	"strings"
	"sync"

	"github.com/N1k3YB/GiftScheduleBot/config"
	"github.com/N1k3YB/GiftScheduleBot/db"
	tele "gopkg.in/telebot.v3"
)

var (
	awaitBroadcast = make(map[int64]bool)
	awaitSearch    = make(map[int64]bool)
	awaitMu        sync.Mutex
)

func handleAdmin(c tele.Context) error {
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return nil
	}
	return showAdminMenu(c)
}

func showAdminMenu(c tele.Context) error {
	text := "🔧 <b>Панель администратора</b>\n\nВыбери раздел:"
	menu := B.NewMarkup()
	menu.Inline(
		menu.Row(menu.Data("📊 Статистика", "admin_stats")),
		menu.Row(
			menu.Data("👥 Пользователи", "admin_users:0"),
			menu.Data("📋 Розыгрыши", "admin_posts:0"),
		),
		menu.Row(
			menu.Data("📢 Рассылка", "admin_broadcast"),
			menu.Data("🔍 Поиск юзера", "admin_search"),
		),
		menu.Row(menu.Data("❌ Закрыть", "close")),
	)
	return editOrSend(c, text, menu)
}

func adminStats(c tele.Context) error {
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return c.Respond()
	}
	_ = c.Respond()

	uStats, err := db.GetUserStats()
	if err != nil {
		return editOrSend(c, "❌ Ошибка загрузки статистики.", nil)
	}
	total, active, completed, newToday, newWeek, err := db.GetPostStats()
	if err != nil {
		return editOrSend(c, "❌ Ошибка загрузки статистики постов.", nil)
	}

	text := fmt.Sprintf(
		"📊 <b>Статистика бота</b>\n\n"+
			"👥 Пользователи:\n"+
			"  Всего: <b>%d</b>\n"+
			"  Новых сегодня: <b>%d</b>\n"+
			"  Новых за неделю: <b>%d</b>\n\n"+
			"🎁 Розыгрыши:\n"+
			"  Всего: <b>%d</b>\n"+
			"  Активных: <b>%d</b>\n"+
			"  Завершённых: <b>%d</b>\n"+
			"  Новых сегодня: <b>%d</b>\n"+
			"  Новых за неделю: <b>%d</b>",
		uStats.TotalUsers, uStats.NewToday, uStats.NewThisWeek,
		total, active, completed, newToday, newWeek,
	)

	menu := B.NewMarkup()
	menu.Inline(menu.Row(menu.Data("◀️ Назад", "admin_menu")))
	return editOrSend(c, text, menu)
}

func adminUsers(c tele.Context, page int) error {
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return c.Respond()
	}
	_ = c.Respond()

	const perPage = 10
	total, _ := db.CountUsers()
	if total == 0 {
		menu := B.NewMarkup()
		menu.Inline(menu.Row(menu.Data("◀️ Назад", "admin_menu")))
		return editOrSend(c, "👥 Пользователей нет.", menu)
	}

	totalPages := (total + perPage - 1) / perPage
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	users, err := db.ListUsers(perPage, page*perPage)
	if err != nil {
		return editOrSend(c, "❌ Ошибка.", nil)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👥 <b>Пользователи</b> (стр. %d/%d)\n\n", page+1, totalPages))
	for i, u := range users {
		num := page*perPage + i + 1
		uname := u.FirstName
		if u.Username != "" {
			uname += " @" + u.Username
		}
		banned := ""
		if u.IsBanned {
			banned = " 🚫"
		}
		sb.WriteString(fmt.Sprintf("%d. %s [%d]%s\n", num, escapeHTML(uname), u.TelegramID, banned))
	}

	menu := B.NewMarkup()
	var rows []tele.Row
	for _, u := range users {
		uname := u.FirstName
		if len(uname) > 15 {
			uname = uname[:15]
		}
		rows = append(rows, menu.Row(menu.Data(
			fmt.Sprintf("👤 %s", uname),
			fmt.Sprintf("admin_user:%d", u.ID),
		)))
	}
	rows = append(rows, buildNavRow(menu, "admin_users", page, totalPages))
	rows = append(rows, menu.Row(menu.Data("◀️ Назад", "admin_menu")))
	menu.Inline(rows...)

	return editOrSend(c, sb.String(), menu)
}

func adminUserDetail(c tele.Context, userID int64) error {
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return c.Respond()
	}
	_ = c.Respond()

	u, err := db.GetUserByID(userID)
	if err != nil {
		return editOrSend(c, "❌ Пользователь не найден.", nil)
	}

	total, _ := db.CountUserPosts(u.ID)
	active, _ := db.CountUserActivePosts(u.ID)

	uname := u.FirstName + " " + u.LastName
	if u.Username != "" {
		uname += " @" + u.Username
	}

	text := fmt.Sprintf(
		"👤 <b>%s</b>\n\nTelegram ID: <code>%d</code>\nРегистрация: %s\n\nРозыгрышей: %d (активных: %d)\nЗаблокирован: %v",
		escapeHTML(uname), u.TelegramID,
		u.CreatedAt.Format("02.01.2006 15:04"),
		total, active, u.IsBanned,
	)

	menu := B.NewMarkup()
	if u.IsBanned {
		menu.Inline(
			menu.Row(menu.Data("✅ Разблокировать", fmt.Sprintf("admin_unban:%d", userID))),
			menu.Row(menu.Data("◀️ Назад", "admin_users:0")),
		)
	} else {
		menu.Inline(
			menu.Row(menu.Data("🚫 Заблокировать", fmt.Sprintf("admin_ban:%d", userID))),
			menu.Row(menu.Data("◀️ Назад", "admin_users:0")),
		)
	}
	return editOrSend(c, text, menu)
}

func adminBanUser(c tele.Context, userID int64, ban bool) error {
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return c.Respond()
	}
	_ = c.Respond()

	if err := db.SetBanned(userID, ban); err != nil {
		return editOrSend(c, "❌ Ошибка.", nil)
	}
	action := "заблокирован"
	if !ban {
		action = "разблокирован"
	}
	return editOrSend(c, fmt.Sprintf("✅ Пользователь %s.", action), func() *tele.ReplyMarkup {
		m := B.NewMarkup()
		m.Inline(m.Row(m.Data("◀️ Назад", fmt.Sprintf("admin_user:%d", userID))))
		return m
	}())
}

func adminPosts(c tele.Context, page int) error {
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return c.Respond()
	}
	_ = c.Respond()

	const perPage = 10
	total, _ := db.CountAllPosts()
	if total == 0 {
		menu := B.NewMarkup()
		menu.Inline(menu.Row(menu.Data("◀️ Назад", "admin_menu")))
		return editOrSend(c, "📋 Розыгрышей нет.", menu)
	}

	totalPages := (total + perPage - 1) / perPage
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	posts, err := db.GetAllPosts(perPage, page*perPage)
	if err != nil {
		return editOrSend(c, "❌ Ошибка.", nil)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 <b>Все розыгрыши</b> (стр. %d/%d)\n\n", page+1, totalPages))
	for i, p := range posts {
		num := page*perPage + i + 1
		status := ""
		if p.IsCompleted {
			status = " ✅"
		}
		title := p.Title
		if title == "" {
			title = "Без названия"
		}
		sb.WriteString(fmt.Sprintf("%d. #%d %s%s\n", num, p.ID, escapeHTML(title), status))
	}

	menu := B.NewMarkup()
	var rows []tele.Row
	for _, p := range posts {
		title := p.Title
		if len(title) > 15 {
			title = title[:15] + "…"
		}
		rows = append(rows, menu.Row(menu.Data(
			fmt.Sprintf("#%d %s", p.ID, title),
			fmt.Sprintf("admin_post:%d", p.ID),
		)))
	}
	rows = append(rows, buildNavRow(menu, "admin_posts", page, totalPages))
	rows = append(rows, menu.Row(menu.Data("◀️ Назад", "admin_menu")))
	menu.Inline(rows...)

	return editOrSend(c, sb.String(), menu)
}

func adminPostDetail(c tele.Context, postID int64) error {
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return c.Respond()
	}
	_ = c.Respond()

	p, err := db.GetPostByID(postID)
	if err != nil {
		return editOrSend(c, "❌ Пост не найден.", nil)
	}

	title := p.Title
	if title == "" {
		title = "Без названия"
	}
	endStr := "—"
	if p.EndDate != nil {
		endStr = p.EndDate.Format("02.01.2006 15:04")
	}
	link := PostLink(p)
	text := fmt.Sprintf(
		"📋 <b>Розыгрыш #%d</b>\n\n"+
			"Название: %s\n"+
			"Канал: %s\n"+
			"Завершается: %s\n"+
			"Завершён: %v\n"+
			"Распарсен: %v",
		p.ID, escapeHTML(title), escapeHTML(p.ChannelUsername),
		endStr, p.IsCompleted, p.ContentParsed,
	)
	if link != "" {
		text += fmt.Sprintf("\n🔗 <a href=\"%s\">Открыть</a>", link)
	}

	menu := B.NewMarkup()
	var rows []tele.Row
	if !p.IsCompleted {
		rows = append(rows, menu.Row(menu.Data("✅ Завершить", fmt.Sprintf("admin_complete_post:%d", postID))))
	}
	rows = append(rows, menu.Row(menu.Data("🗑 Удалить из БД", fmt.Sprintf("admin_del_post:%d", postID))))
	rows = append(rows, menu.Row(menu.Data("◀️ Назад", "admin_posts:0")))
	menu.Inline(rows...)

	return editOrSend(c, text, menu)
}

func adminDeletePost(c tele.Context, postID int64) error {
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return c.Respond()
	}
	_ = c.Respond()

	if err := db.DeletePostFromDB(postID); err != nil {
		return editOrSend(c, "❌ Ошибка удаления.", nil)
	}
	menu := B.NewMarkup()
	menu.Inline(menu.Row(menu.Data("◀️ Список постов", "admin_posts:0")))
	return editOrSend(c, fmt.Sprintf("✅ Розыгрыш #%d удалён из БД.", postID), menu)
}

func adminCompletePost(c tele.Context, postID int64) error {
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return c.Respond()
	}
	_ = c.Respond()

	if err := db.MarkCompleted(postID); err != nil {
		return editOrSend(c, "❌ Ошибка.", nil)
	}
	return adminPostDetail(c, postID)
}

func adminBroadcastPrompt(c tele.Context) error {
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return c.Respond()
	}
	_ = c.Respond()

	awaitMu.Lock()
	awaitBroadcast[sender.ID] = true
	awaitMu.Unlock()

	menu := B.NewMarkup()
	menu.Inline(menu.Row(menu.Data("❌ Отмена", "admin_menu")))
	return editOrSend(c, "📢 Введи текст рассылки следующим сообщением:", menu)
}

func adminSearchPrompt(c tele.Context) error {
	sender := c.Sender()
	if sender == nil || !config.IsAdmin(sender.ID) {
		return c.Respond()
	}
	_ = c.Respond()

	awaitMu.Lock()
	awaitSearch[sender.ID] = true
	awaitMu.Unlock()

	menu := B.NewMarkup()
	menu.Inline(menu.Row(menu.Data("❌ Отмена", "admin_menu")))
	return editOrSend(c, "🔍 Введи username или Telegram ID:", menu)
}

func HandleAdminText(c tele.Context, senderID int64) (handled bool) {
	awaitMu.Lock()
	isBroadcast := awaitBroadcast[senderID]
	isSearch := awaitSearch[senderID]
	awaitMu.Unlock()

	if isBroadcast {
		awaitMu.Lock()
		delete(awaitBroadcast, senderID)
		awaitMu.Unlock()
		return handleBroadcast(c, c.Message().Text)
	}
	if isSearch {
		awaitMu.Lock()
		delete(awaitSearch, senderID)
		awaitMu.Unlock()
		return handleSearch(c, c.Message().Text)
	}
	return false
}

func handleBroadcast(c tele.Context, text string) bool {
	users, err := db.GetAllUsersForBroadcast()
	if err != nil {
		_ = c.Send("❌ Ошибка получения пользователей.")
		return true
	}
	sent, failed := 0, 0
	for _, u := range users {
		_, err := B.Send(&tele.Chat{ID: u.TelegramID}, text, &tele.SendOptions{ParseMode: tele.ModeHTML})
		if err != nil {
			failed++
		} else {
			sent++
		}
	}
	_ = c.Send(fmt.Sprintf("✅ Рассылка завершена.\nОтправлено: %d\nОшибок: %d", sent, failed))
	return true
}

func handleSearch(c tele.Context, query string) bool {
	users, err := db.SearchUser(strings.TrimSpace(query))
	if err != nil || len(users) == 0 {
		_ = c.Send("🔍 Пользователи не найдены.")
		return true
	}
	var sb strings.Builder
	sb.WriteString("🔍 <b>Результаты поиска:</b>\n\n")
	for _, u := range users {
		uname := u.FirstName + " " + u.LastName
		if u.Username != "" {
			uname += " @" + u.Username
		}
		banned := ""
		if u.IsBanned {
			banned = " 🚫"
		}
		sb.WriteString(fmt.Sprintf("• %s [<code>%d</code>]%s\n", escapeHTML(uname), u.TelegramID, banned))
	}
	_ = c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML})
	return true
}
