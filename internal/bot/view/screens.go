package view

import (
	"fmt"
	"html"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"locatorbpla/internal/domain"
)

const donateURL = "https://pay.cloudtips.ru/p/8d8b98bb"

const pageSize = 5

func Welcome() (string, tgbotapi.InlineKeyboardMarkup) {
	text := `Добро пожаловать в <b>Локатор России</b>! 🛡️

` + `Это ваш персональный помощник для отслеживания угрозы БПЛА в режиме реального времени.

` + `<b>Как это работает:</b>
` + `• Бот мониторит указанные вами <b>локации</b>.
` + `• При появлении опасности вы мгновенно получаете <b>уведомление</b>.

` + `<b>Для начала работы:</b>
` + `1. Перейдите в <b>Настройки</b> (кнопка ниже).
` + `2. Укажите регионы, которые хотите отслеживать.
` + `<i>Будьте в курсе. Оставайтесь в безопасности.</i>

` + `<b>Доступные команды:</b>
` + `• <code>/start</code> — показать это меню
` + `• <code>/settings</code> — открыть настройки

` + `<b>Основная кнопка:</b>
` + `⚙️ <b>Настройки</b> — выбрать локации и управлять уведомлениями.`
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "get-settings"),
		),
	)
	return text, kb
}

func Settings(user domain.User) (string, tgbotapi.InlineKeyboardMarkup) {
	locs := html.EscapeString(clip(strings.Join(user.Locations, ", "), 400))
	notifMark := "🔕"
	notifState := "отключены"
	toggleText := "🔔 Уведомления"
	toggleData := "set-notifications-true"
	toggleHint := "включить"
	if user.Notifications {
		notifMark = "🔔"
		notifState = "включены"
		toggleText = "🔕 Уведомления"
		toggleData = "set-notifications-false"
		toggleHint = "отключить"
	}
	text := fmt.Sprintf(`⚙️ <b>Настройки</b>

Здесь вы можете настроить параметры получения уведомлений о БПЛА.

<b>Текущие настройки:</b>
📍 <b>Отслеживаемые локации</b>: <blockquote expandable><code>%s</code></blockquote>
%s <b>Уведомления</b>: %s

<b>Что вы можете изменить:</b>
• Выбрать конкретные регионы для отслеживания угроз
• Включить или отключить получение уведомлений
• Изменить список локаций в любое время

<b>Кнопки настроек:</b>
📍 <b>Локации</b> — изменить список отслеживаемых регионов
%s <b>Уведомления</b> — %s оповещения о новых угрозах
❤️ <b>Поддержать проект</b> — поддержать проект чаевыми по ссылке
`, locs, notifMark, notifState, notifMark, toggleHint)

	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📍 Локации", "set-locations")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(toggleText, toggleData)),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("❤️ Поддержать проект", donateURL)),
	}
	if user.Admin {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛠️ Админка", "get-admin"),
		))
	}
	return text, tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func LocationsPrompt() (string, tgbotapi.InlineKeyboardMarkup) {
	text := `📍 <b>Управление локациями</b>

Введите названия регионов, городов или областей для отслеживания через запятую.

<b>Формат ввода:</b>
<code>Москва, Санкт-Петербург, Краснодарский край, Белгородская область</code>

<b>Принцип работы:</b>
Уведомление будет приходить, если в оповещении будет упомянута <i>хотя бы одна</i> из указанных вами локаций.
То есть, поиск работает по принципу <b>"ИЛИ"</b> (OR).

<b>Настоятельно</b> рекомендуется использовать <b>сокращенное название</b> локации, чтобы поиск был более точным.
<b>Пример:</b> Вместо "Краснодарскии краи" используйте "Краснодар".
<b>Пример использования:</b>
Если вы укажете: "Москва, Калуга"
• Вы получите уведомление, если будет угроза в <b>Москве</b>
• Вы получите уведомление, если будет угроза в <b>Калуге</b>
• Вы получите уведомление, если будет угроза в <b>Москве и Калуге</b>

Введите новые локации, чтобы заменить текущий список, или нажмите <b>Отмена</b> для возврата.`
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel-locations"),
		),
	)
	return text, kb
}

func ErrorPrompt() (string, tgbotapi.InlineKeyboardMarkup) {
	text := `⚠️ <b>Опишите ошибку, с которой вы столкнулись:</b>

Пожалуйста, подробно опишите, что именно пошло не так.
Укажите, на каком этапе произошла ошибка, что вы делали перед этим и что именно не сработало.
Чем подробнее будет ваш ответ — тем быстрее мы сможем всё исправить.`
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", "cancel-error"),
		),
	)
	return text, kb
}

func ErrorThanks() (string, tgbotapi.InlineKeyboardMarkup) {
	text := `🙏 <b>Спасибо за обратную связь!</b>

Мы посмотрим сообщение и поправим, если это ошибка бота.
Если появится ещё что-то — напишите через /error.

<b>Чтобы продолжить:</b>
⚙️ <b>Настройки</b> — локации и уведомления.`
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "get-settings"),
		),
	)
	return text, kb
}

func AdminHome(total int) (string, tgbotapi.InlineKeyboardMarkup) {
	text := fmt.Sprintf("Панель администратора:\n\n👥 <b>Всего пользователей</b>: %d", total)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Получить всех пользователей", "get-admin-users"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "get-settings"),
		),
	)
	return text, kb
}

func AdminUsers(users []domain.User, page int) (string, tgbotapi.InlineKeyboardMarkup) {
	pages := 1
	if n := len(users); n > 0 {
		pages = (n + pageSize - 1) / pageSize
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start := page * pageSize
	end := start + pageSize
	if end > len(users) {
		end = len(users)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Пользователи, страница %d/%d\n\n", page+1, pages)
	var rows [][]tgbotapi.InlineKeyboardButton
	if start < end {
		for _, user := range users[start:end] {
			name := user.Username
			if name == "" {
				name = user.Name
			}
			flag := "❌"
			if user.Admin {
				flag = "✅"
			}
			locs := clip(strings.Join(user.Locations, ", "), 80)
			fmt.Fprintf(&b, "• @%s <code>%d</code> админ %s\n🗺️ %s\n\n", html.EscapeString(name), user.TgID, flag, html.EscapeString(locs))
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"Админ: "+clip(name, 20),
					fmt.Sprintf("set-admin-%d-%d", user.TgID, page),
				),
			))
		}
	} else {
		b.WriteString("Пользователей нет.")
	}
	var nav []tgbotapi.InlineKeyboardButton
	if page > 0 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("◀️", fmt.Sprintf("get-admin-users-%d", page-1)))
	}
	if page+1 < pages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("▶️", fmt.Sprintf("get-admin-users-%d", page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, nav)
	}
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Назад", "get-admin"),
	))
	return b.String(), tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
