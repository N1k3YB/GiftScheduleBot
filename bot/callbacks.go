package bot

import (
	"fmt"
	"strings"

	"github.com/nkeydash/GiftScheduleBot/db"
	"github.com/nkeydash/GiftScheduleBot/parser"
	tele "gopkg.in/telebot.v3"
)

func handleCallback(c tele.Context) error {
	cb := c.Callback()
	if cb == nil {
		return nil
	}
	data := cb.Data
	if strings.HasPrefix(data, "\f") {
		data = data[1:]
	}

	parts := strings.SplitN(data, ":", 2)
	action := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}

	switch action {
	case "noop":
		return c.Respond()
	case "close":
		_ = c.Respond()
		return c.Delete()
	case "main_menu":
		_ = c.Respond()
		return showMainMenu(c)
	case "my_list":
		_ = c.Respond()
		u := getDBUser(c)
		if u == nil {
			return ensureAuth(c, func(u *db.User) error {
				return showMyList(c, u, int(parseInt64(arg)))
			})
		}
		return showMyList(c, u, int(parseInt64(arg)))
	case "all_list":
		_ = c.Respond()
		return showAllList(c, int(parseInt64(arg)))
	case "del_my":
		_ = c.Respond()
		return deleteMyPost(c, parseInt64(arg))
	case "profile":
		_ = c.Respond()
		return showProfile(c)
	case "notify_toggle":
		_ = c.Respond()
		return toggleNotify(c, arg)
	case "check_result":
		_ = c.Respond()
		return checkResult(c, parseInt64(arg))

	case "admin_stats":
		return adminStats(c)
	case "admin_users":
		return adminUsers(c, int(parseInt64(arg)))
	case "admin_user":
		return adminUserDetail(c, parseInt64(arg))
	case "admin_ban":
		return adminBanUser(c, parseInt64(arg), true)
	case "admin_unban":
		return adminBanUser(c, parseInt64(arg), false)
	case "admin_posts":
		return adminPosts(c, int(parseInt64(arg)))
	case "admin_post":
		return adminPostDetail(c, parseInt64(arg))
	case "admin_del_post":
		return adminDeletePost(c, parseInt64(arg))
	case "admin_complete_post":
		return adminCompletePost(c, parseInt64(arg))
	case "admin_broadcast":
		return adminBroadcastPrompt(c)
	case "admin_search":
		return adminSearchPrompt(c)
	case "admin_menu":
		return showAdminMenu(c)
	}
	return c.Respond()
}

func ensureAuth(c tele.Context, fn func(*db.User) error) error {
	sender := c.Sender()
	if sender == nil {
		return nil
	}
	u, err := db.UpsertUser(sender.ID, sender.Username, sender.FirstName, sender.LastName)
	if err != nil || u == nil || u.IsBanned {
		return nil
	}
	return fn(u)
}

func showMainMenu(c tele.Context) error {
	sender := c.Sender()
	var name string
	if sender != nil {
		name = sender.FirstName
	}
	text := fmt.Sprintf(
		"👋 <b>%s</b>, добро пожаловать!\n\n"+
			"📂 <b>Общий список</b> формируется из розыгрышей, которые сохранили пользователи бота.\n\n"+
			"Выбери действие 👇",
		name,
	)
	return editOrSend(c, text, mainMenuMarkup())
}

func showAllList(c tele.Context, page int) error {
	const perPage = 10
	total, _ := db.CountAllPosts()
	if total == 0 {
		menu := B.NewMarkup()
		menu.Inline(
			menu.Row(menu.Data("🏠 Главное меню", "main_menu")),
			menu.Row(menu.Data("❌ Закрыть", "close")),
		)
		return editOrSend(c, "📭 Общий список пока пуст. Добавь первый розыгрыш!", menu)
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
		return editOrSend(c, "❌ Ошибка загрузки.", nil)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🌐 <b>Все розыгрыши</b> (стр. %d/%d)\n\n", page+1, totalPages))
	for i, p := range posts {
		num := page*perPage + i + 1
		sb.WriteString(fmt.Sprintf("%d. %s\n", num, PostShortText(p)))
	}

	menu := B.NewMarkup()
	navRow := buildNavRow(menu, "all_list", page, totalPages)
	menu.Inline(
		navRow,
		menu.Row(menu.Data("🏠 Назад", "main_menu")),
		menu.Row(menu.Data("❌ Закрыть", "close")),
	)

	return editOrSend(c, sb.String(), menu)
}

func deleteMyPost(c tele.Context, userPostID int64) error {
	sender := c.Sender()
	if sender == nil {
		return c.Respond()
	}
	u, err := db.GetUserByTelegramID(sender.ID)
	if err != nil {
		return c.Respond()
	}
	if err := db.RemoveUserPost(u.ID, userPostID); err != nil {
		return c.Edit("❌ Не удалось удалить.")
	}
	return showMyList(c, u, 0)
}

func showProfile(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Respond()
	}
	u, err := db.GetUserByTelegramID(sender.ID)
	if err != nil {
		return c.Respond()
	}

	total, _ := db.CountUserPosts(u.ID)
	active, _ := db.CountUserActivePosts(u.ID)
	completed, _ := db.CountUserCompletedPosts(u.ID)

	uname := ""
	if u.Username != "" {
		uname = "@" + u.Username
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👤 <b>Профиль</b>\n\n"))
	sb.WriteString(fmt.Sprintf("Имя: <b>%s %s</b>\n", escapeHTML(u.FirstName), escapeHTML(u.LastName)))
	if uname != "" {
		sb.WriteString(fmt.Sprintf("Username: <b>%s</b>\n", uname))
	}
	sb.WriteString(fmt.Sprintf("Дата регистрации: <b>%s</b>\n\n", u.CreatedAt.Format("02.01.2006")))
	sb.WriteString(fmt.Sprintf("🎁 Розыгрышей всего: <b>%d</b>\n", total))
	sb.WriteString(fmt.Sprintf("  • Активных: <b>%d</b>\n", active))
	sb.WriteString(fmt.Sprintf("  • Завершённых: <b>%d</b>\n\n", completed))
	sb.WriteString("🔔 <b>Уведомления:</b>\n")
	sb.WriteString(fmt.Sprintf("  %s За 3 дня\n", notifyIcon(u.Notify3Days)))
	sb.WriteString(fmt.Sprintf("  %s За 1 день\n", notifyIcon(u.Notify1Day)))
	sb.WriteString(fmt.Sprintf("  %s За 1 час\n", notifyIcon(u.Notify1Hour)))
	sb.WriteString("  🔔 Итоги розыгрыша — всегда\n")

	menu := B.NewMarkup()
	n3icon := notifyIcon(u.Notify3Days)
	n1icon := notifyIcon(u.Notify1Day)
	nhicon := notifyIcon(u.Notify1Hour)
	menu.Inline(
		menu.Row(
			menu.Data(fmt.Sprintf("%s За 3 дня", n3icon), "notify_toggle:notify_3days"),
			menu.Data(fmt.Sprintf("%s За 1 день", n1icon), "notify_toggle:notify_1day"),
			menu.Data(fmt.Sprintf("%s За 1 час", nhicon), "notify_toggle:notify_1hour"),
		),
		menu.Row(menu.Data("📋 Мои розыгрыши", "my_list:0")),
		menu.Row(menu.Data("🏠 Главное меню", "main_menu")),
		menu.Row(menu.Data("❌ Закрыть", "close")),
	)

	return editOrSend(c, sb.String(), menu)
}

func notifyIcon(on bool) string {
	if on {
		return "🔔"
	}
	return "🔕"
}

func toggleNotify(c tele.Context, field string) error {
	sender := c.Sender()
	if sender == nil {
		return c.Respond()
	}
	u, err := db.GetUserByTelegramID(sender.ID)
	if err != nil {
		return c.Respond()
	}
	var current bool
	switch field {
	case "notify_3days":
		current = u.Notify3Days
	case "notify_1day":
		current = u.Notify1Day
	case "notify_1hour":
		current = u.Notify1Hour
	default:
		return c.Respond()
	}
	_ = db.SetNotify(u.ID, field, !current)
	return showProfile(c)
}

func checkResult(c tele.Context, postID int64) error {
	sender := c.Sender()
	if sender == nil {
		return c.Respond()
	}
	u, err := db.GetUserByTelegramID(sender.ID)
	if err != nil {
		return c.Respond()
	}

	p, err := db.GetPostByID(postID)
	if err != nil {
		return editOrSend(c, "❌ Розыгрыш не найден.", nil)
	}

	if !p.IsCompleted {
		menu := B.NewMarkup()
		menu.Inline(menu.Row(menu.Data("◀️ Назад", "my_list:0")))
		msg := "⏳ Розыгрыш ещё не завершён."
		if p.EndDate != nil {
			msg += fmt.Sprintf("\nОжидаемая дата завершения: <b>%s</b>", p.EndDate.Format("02.01.2006 15:04"))
		}
		return editOrSend(c, msg, menu)
	}

	var resultText string
	if p.ResultsInSamePost || p.ResultsURL == "" {
		resultText = p.RawText
	}

	result := parser.CheckWinner(resultText, u.Username, u.FirstName, u.LastName, p.EndDate)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎰 <b>%s</b>\n\n", escapeHTML(p.Title)))
	sb.WriteString(result.Message + "\n")

	link := PostLink(p)
	if link != "" {
		sb.WriteString(fmt.Sprintf("\n🔗 <a href=\"%s\">Открыть пост</a>", link))
	}
	if p.ResultsURL != "" && !p.ResultsInSamePost {
		sb.WriteString(fmt.Sprintf("\n📋 <a href=\"%s\">Пост с итогами</a>", p.ResultsURL))
	}

	menu := B.NewMarkup()
	menu.Inline(menu.Row(menu.Data("◀️ Мои розыгрыши", "my_list:0")))
	return editOrSend(c, sb.String(), menu)
}
