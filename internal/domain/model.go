package domain

import "errors"

const (
	WaitLocations = "locations"
	WaitError     = "error"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID            int64
	TgID          int64
	Admin         bool
	Notifications bool
	Name          string
	Username      string
	Locations     []string
}

type Session struct {
	TgID       int64
	WaitingFor string
	MsgID      int
	ErrorMsgID int
}

func WaitingLabel(waiting string) string {
	switch waiting {
	case WaitLocations:
		return "список локаций"
	case WaitError:
		return "описание ошибки"
	default:
		return "текст"
	}
}
