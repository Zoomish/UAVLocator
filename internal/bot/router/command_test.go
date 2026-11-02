package router

import (
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestCommandName(t *testing.T) {
	tests := []struct {
		text string
		want string
		ok   bool
	}{
		{text: "/start", want: "start", ok: true},
		{text: "/Start", want: "start", ok: true},
		{text: "/start@LocatorBot", want: "start", ok: true},
		{text: "/settings now", want: "settings", ok: true},
		{text: "hello"},
		{text: "see /start later"},
	}
	for _, tt := range tests {
		msg := &models.Message{Text: tt.text}
		if tt.ok || tt.text != "" {
			msg.Entities = []models.MessageEntity{{
				Type:   models.MessageEntityTypeBotCommand,
				Offset: 0,
				Length: commandLen(tt.text),
			}}
		}
		if tt.text == "hello" || tt.text == "see /start later" {
			msg.Entities = nil
		}
		if tt.text == "see /start later" {
			msg.Entities = []models.MessageEntity{{
				Type:   models.MessageEntityTypeBotCommand,
				Offset: 4,
				Length: 6,
			}}
		}
		got, ok := commandName(msg)
		if ok != tt.ok || got != tt.want {
			t.Fatalf("%q: got %q %v, want %q %v", tt.text, got, ok, tt.want, tt.ok)
		}
	}
}

func commandLen(text string) int {
	n := 0
	for _, r := range text {
		if r == ' ' {
			break
		}
		n++
	}
	return n
}
