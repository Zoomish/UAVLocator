package view

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-telegram/bot/models"

	"locatorbpla/internal/domain"
)

const donateURL = "https://pay.cloudtips.ru/p/8d8b98bb"

const pageSize = 5

func Welcome() (string, models.InlineKeyboardMarkup) {
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
	kb := markup(row(dataBtn("⚙️ Настройки", "get-settings")))
	return text, kb
}

func GroupBoard(user domain.User) (string, models.InlineKeyboardMarkup) {
	locs := "не заданы"
	if len(user.Locations) > 0 {
		locs = html.EscapeString(clip(strings.Join(user.Locations, ", "), 400))
	}
	editor := "ещё никто"
	if label := editorLabel(user); label != "" {
		editor = label
	}
	text := fmt.Sprintf(`<b>Локатор в этом чате</b>

Локации: %s
Менял: %s

Тревога приходит одним сообщением в чат. Настройки видит только тот, кто нажал кнопку.`, locs, editor)
	kb := markup(row(dataBtn("⚙️ Настройки", "get-settings")))
	return text, kb
}

func OnlyYou(name string) string {
	who := strings.TrimSpace(name)
	if who == "" {
		who = "вы"
	}
	return "Это сообщение видите только вы, " + html.EscapeString(who) + "."
}

func HiddenIntro(name string, askReply bool) string {
	if !askReply {
		return OnlyYou(name) + "\n\n"
	}
	return OnlyYou(name) + "\n\nНе пишите ответ в чат. Нажмите «Ответить» на это сообщение и отправьте текст сюда. Обычное сообщение в чат увидят все, бот его не примет.\n\n"
}

func Settings(user domain.User, private bool) (string, models.InlineKeyboardMarkup) {
	locs := html.EscapeString(clip(strings.Join(user.Locations, ", "), 400))
	editorLine := ""
	if label := editorLabel(user); label != "" {
		editorLine = "\nМенял: " + label
	}
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
📍 <b>Отслеживаемые локации</b>: <blockquote expandable><code>%s</code></blockquote>%s
%s <b>Уведомления</b>: %s

<b>Что вы можете изменить:</b>
• Выбрать конкретные регионы для отслеживания угроз
• Включить или отключить получение уведомлений
• Изменить список локаций в любое время

<b>Кнопки настроек:</b>
📍 <b>Локации</b> — изменить список отслеживаемых регионов
%s <b>Уведомления</b> — %s оповещения о новых угрозах
❤️ <b>Поддержать проект</b> — поддержать проект чаевыми по ссылке
`, locs, editorLine, notifMark, notifState, notifMark, toggleHint)

	rows := [][]models.InlineKeyboardButton{
		row(dataBtn("📍 Локации", "set-locations")),
		row(dataBtn(toggleText, toggleData)),
		row(urlBtn("❤️ Поддержать проект", donateURL)),
	}
	if private {
		rows = append(rows, row(dataBtn("API", "get-api")))
	}
	if private && user.Admin {
		rows = append(rows, row(dataBtn("🛠️ Админка", "get-admin")))
	}
	return text, markup(rows...)
}

func LocationsPrompt() (string, models.InlineKeyboardMarkup) {
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
	kb := markup(row(dataBtn("❌ Отмена", "cancel-locations")))
	return text, kb
}

func APIMenu(user domain.User) (string, models.InlineKeyboardMarkup) {
	state := "не задан"
	if user.WebhookURL != "" {
		state = html.EscapeString(endpoint(user.WebhookURL))
	}
	text := fmt.Sprintf(`<b>API</b>

Сейчас: <code>%s</code>

После тревоги бот отправляет POST на этот адрес и не читает ответ.`, state)
	kb := markup(
		row(dataBtn("Изменить", "set-api")),
		row(dataBtn("Отключить", "set-api-off")),
	)
	return text, kb
}

func APIPrompt(group bool) (string, models.InlineKeyboardMarkup) {
	text := `Пришлите один адрес <code>https://</code>. После тревоги бот отправит туда POST и не будет ждать ответ.

<code>-</code> стирает адрес.`
	if group {
		text = `Не пишите адрес в чат. Нажмите «Ответить» именно на это сообщение и пришлите один адрес https.

Обычное сообщение в чат увидят все, бот его не сохранит.`
	}
	kb := markup(row(dataBtn("❌ Отмена", "cancel-api")))
	return text, kb
}

func Confirm(title, phrase, question, save string) (string, models.InlineKeyboardMarkup) {
	text := fmt.Sprintf(`<b>%s</b>

<blockquote expandable><code>%s</code></blockquote>

%s`, html.EscapeString(title), html.EscapeString(phrase), html.EscapeString(question))
	kb := markup(
		row(dataBtn(save, "confirm-save")),
		row(dataBtn("✏️ Ввести заново", "confirm-again")),
	)
	return text, kb
}

func endpoint(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "задан"
	}
	if port := u.Port(); port != "" {
		return u.Hostname() + ":" + port
	}
	return u.Hostname()
}

func editorLabel(user domain.User) string {
	if user.LocationsEditor != "" {
		return html.EscapeString(user.LocationsEditor)
	}
	if user.LocationsEditorID != 0 {
		return strconv.FormatInt(user.LocationsEditorID, 10)
	}
	return ""
}

func ErrorPrompt() (string, models.InlineKeyboardMarkup) {
	text := `⚠️ <b>Опишите ошибку, с которой вы столкнулись:</b>

Пожалуйста, подробно опишите, что именно пошло не так.
Укажите, на каком этапе произошла ошибка, что вы делали перед этим и что именно не сработало.
Чем подробнее будет ваш ответ — тем быстрее мы сможем всё исправить.`
	kb := markup(row(dataBtn("❌ Отменить", "cancel-error")))
	return text, kb
}

func ErrorThanks() (string, models.InlineKeyboardMarkup) {
	text := `🙏 <b>Спасибо за обратную связь!</b>

Мы посмотрим сообщение и поправим, если это ошибка бота.
Если появится ещё что-то — напишите через /error.

<b>Чтобы продолжить:</b>
⚙️ <b>Настройки</b> — локации и уведомления.`
	kb := markup(row(dataBtn("⚙️ Настройки", "get-settings")))
	return text, kb
}

func AdminHome(audience domain.Audience, post domain.PostStats) (string, models.InlineKeyboardMarkup) {
	text := fmt.Sprintf(`Панель администратора

📍 <b>Без локаций</b>: %d
🔕 <b>Сами выключили уведомления</b>: %d
🚫 <b>Заблокировали бота</b>: %d

%s`, audience.NoLocations, audience.Muted, audience.Blocked, lastPost(post))
	kb := markup(
		row(dataBtn("Получить всех пользователей", "get-admin-users")),
		row(dataBtn("⚙️ Настройки", "get-settings")),
	)
	return text, kb
}

func lastPost(post domain.PostStats) string {
	if !post.Present {
		return "<b>Последний пост</b>\nЕщё не было."
	}
	return fmt.Sprintf(`<b>Последний пост</b>
✅ Получили: %d
🚫 Заблокировали на рассылке: %d
⏳ Упёрлись в 429: %d
❌ Так и не ушли: %d`, post.Sent, post.Blocked, post.FloodWaits, post.Undelivered)
}

func AdminUsers(users []domain.User, page int) (string, models.InlineKeyboardMarkup) {
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
	var rows [][]models.InlineKeyboardButton
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
			rows = append(rows, row(dataBtn(
				"Админ: "+clip(name, 20),
				fmt.Sprintf("set-admin-%d-%d", user.TgID, page),
			)))
		}
	} else {
		b.WriteString("Пользователей нет.")
	}
	var nav []models.InlineKeyboardButton
	if page > 0 {
		nav = append(nav, dataBtn("◀️", fmt.Sprintf("get-admin-users-%d", page-1)))
	}
	if page+1 < pages {
		nav = append(nav, dataBtn("▶️", fmt.Sprintf("get-admin-users-%d", page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, nav)
	}
	rows = append(rows, row(dataBtn("Назад", "get-admin")))
	return b.String(), markup(rows...)
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func dataBtn(text, data string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: text, CallbackData: data}
}

func urlBtn(text, link string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{Text: text, URL: link}
}

func row(buttons ...models.InlineKeyboardButton) []models.InlineKeyboardButton {
	return buttons
}

func markup(rows ...[]models.InlineKeyboardButton) models.InlineKeyboardMarkup {
	return models.InlineKeyboardMarkup{InlineKeyboard: rows}
}
