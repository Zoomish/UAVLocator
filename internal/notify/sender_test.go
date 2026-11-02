package notify

import (
	"html"
	"strings"
	"testing"
)

func TestMessage(t *testing.T) {
	got := Message("белгород", "<b>текст</b>")
	want := "совпало: <b>белгород</b>\n\n<b>текст</b>"
	if got != want {
		t.Fatalf("got %q", got)
	}
	got = Message("а<б>", "пост")
	want = "совпало: <b>а&lt;б&gt;</b>\n\nпост"
	if got != want {
		t.Fatalf("got %q", got)
	}
	if Message("", "пост") != "пост" {
		t.Fatal("empty pattern")
	}
}

func TestFitMessage(t *testing.T) {
	short, asHTML := fitMessage("белгород", "текст")
	if !asHTML || short != Message("белгород", "текст") {
		t.Fatalf("short %q html %v", short, asHTML)
	}
	body := "<blockquote expandable>" + html.EscapeString(strings.Repeat("я", 5000)) + "</blockquote>хвост"
	got, asHTML := fitMessage("белгород", body)
	if asHTML {
		t.Fatal("long message kept html")
	}
	if strings.Contains(got, "<") {
		t.Fatalf("tags left %q", got[:80])
	}
	if utf16Len(got) > 4096 {
		t.Fatalf("len %d", utf16Len(got))
	}
	if !strings.HasPrefix(got, "совпало: белгород") {
		t.Fatalf("prefix %q", got[:40])
	}
}
