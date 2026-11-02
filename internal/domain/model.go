package domain

import "errors"

const (
	WaitLocations = "locations"
	WaitError     = "error"
	WaitWebhook   = "webhook"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID                int64
	TgID              int64
	Admin             bool
	Notifications     bool
	Name              string
	Username          string
	Locations         []string
	LocationsEditorID int64
	LocationsEditor   string
	WebhookURL        string
}

type Match struct {
	User    User
	Pattern string
}

type Audience struct {
	People      int
	Active      int
	NoLocations int
	Muted       int
	Blocked     int
}

type PostStats struct {
	Present     bool
	Sent        int
	Blocked     int
	FloodWaits  int
	Undelivered int
}

type Session struct {
	TgID        int64
	WaitingFor  string
	WaitingUser int64
	WaitingName string
	Draft       string
	DraftMsg    int
	MsgID       int
	ErrorMsgID  int
}

func WaitingLabel(waiting string) string {
	switch waiting {
	case WaitLocations:
		return "список локаций"
	case WaitError:
		return "описание ошибки"
	case WaitWebhook:
		return "URL"
	default:
		return "текст"
	}
}
