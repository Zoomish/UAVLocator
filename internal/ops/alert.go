package ops

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"time"

	"github.com/go-telegram/bot"
)

var dialHop = regexp.MustCompile(`(?:\d{1,3}(?:\.\d{1,3}){3}|\[[0-9a-fA-F:]*\]):[0-9]+->(?:\d{1,3}(?:\.\d{1,3}){3}|\[[0-9a-fA-F:]*\]):[0-9]+`)

type note struct {
	text       string
	count      int
	sent       time.Time
	unsent     bool
	quiet      bool
	fromRepeat bool
	hold       string
}

type Alerter struct {
	Bot     *bot.Bot
	AdminID int64
	Log     *slog.Logger
	Every   time.Duration

	once  sync.Once
	mu    sync.Mutex
	notes map[string]*note
	ch    chan string
}

func (a *Alerter) Repeat(ctx context.Context, err error) {
	if err == nil {
		return
	}
	if a.Log != nil {
		a.Log.Error("telegram", "err", err)
	}
	if a.Bot == nil || a.AdminID == 0 {
		return
	}
	a.ready()
	text := clip(stripDial(err.Error()), 3500)
	a.mu.Lock()
	send, summary := repeat(a.notes, text, time.Now(), a.every())
	a.mu.Unlock()
	if summary != "" {
		a.kick(summary)
	}
	if send {
		a.kick(text)
	}
}

func (a *Alerter) Infra(ctx context.Context, err error) {
	if err == nil {
		return
	}
	if a.Log != nil {
		a.Log.Error("infra", "err", err)
	}
	if a.Bot == nil || a.AdminID == 0 {
		return
	}
	a.ready()
	text := clip(err.Error(), 3500)
	a.mu.Lock()
	fresh, summary := stage(a.notes, text, time.Now(), a.every())
	a.mu.Unlock()
	if summary != "" {
		a.kick(summary)
	}
	if fresh {
		a.kick(text)
	}
}

func (a *Alerter) Run(ctx context.Context) error {
	a.ready()
	tick := time.NewTicker(a.every() / 3)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case text := <-a.ch:
			a.try(text)
		case <-tick.C:
			a.flush()
		}
	}
}

func (a *Alerter) ready() {
	a.once.Do(func() {
		if a.notes == nil {
			a.notes = map[string]*note{}
		}
		if a.ch == nil {
			a.ch = make(chan string, 64)
		}
	})
}

func (a *Alerter) every() time.Duration {
	if a.Every > 0 {
		return a.Every
	}
	return 3 * time.Minute
}

func (a *Alerter) kick(text string) {
	select {
	case a.ch <- text:
	default:
	}
}

func (a *Alerter) flush() {
	now := time.Now()
	a.mu.Lock()
	var jobs []string
	for key, n := range a.notes {
		if n.quiet && n.count <= 1 && n.hold == "" && !n.unsent && now.Sub(n.sent) >= a.every() {
			delete(a.notes, key)
			continue
		}
		if n.hold != "" {
			jobs = append(jobs, n.hold)
			continue
		}
		if n.unsent {
			jobs = append(jobs, n.text)
			continue
		}
		if n.count > 1 && now.Sub(n.sent) >= a.every() {
			n.hold = fmt.Sprintf("Повторилась %d раз:\n%s", n.count, n.text)
			n.count = 1
			if n.fromRepeat {
				n.quiet = true
				n.unsent = false
			}
			jobs = append(jobs, n.hold)
		}
	}
	a.mu.Unlock()
	for _, text := range jobs {
		a.try(text)
	}
}

func (a *Alerter) try(text string) {
	if !a.send(text) {
		return
	}
	now := time.Now()
	a.mu.Lock()
	ack(a.notes, text, now)
	a.mu.Unlock()
}

func (a *Alerter) send(text string) bool {
	text = clip(text, 3900)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_, err := a.Bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: a.AdminID,
		Text:   "Ошибка бота:\n" + text,
	})
	if err == nil {
		return true
	}
	if a.Log != nil {
		a.Log.Error("admin notify failed", "err", err)
	}
	return false
}

func stage(notes map[string]*note, text string, now time.Time, every time.Duration) (bool, string) {
	n := notes[text]
	if n == nil {
		notes[text] = &note{text: text, count: 1, sent: now, unsent: true}
		return true, ""
	}
	if n.quiet {
		n.quiet = false
		n.count = 1
		n.sent = now
		n.unsent = true
		return true, ""
	}
	if n.unsent || n.hold != "" || now.Sub(n.sent) < every {
		n.count++
		return false, ""
	}
	summary := ""
	if n.count > 1 {
		summary = fmt.Sprintf("Повторилась %d раз:\n%s", n.count, n.text)
		n.hold = summary
	}
	n.count = 1
	n.sent = now
	n.unsent = true
	return true, summary
}

func repeat(notes map[string]*note, text string, now time.Time, every time.Duration) (bool, string) {
	n := notes[text]
	if n == nil || (!n.unsent && n.hold == "" && now.Sub(n.sent) >= every) {
		summary := ""
		if n != nil && !n.quiet && n.count > 1 {
			summary = fmt.Sprintf("Повторилась %d раз:\n%s", n.count, n.text)
		}
		notes[text] = &note{text: text, count: 1, sent: now, quiet: true, fromRepeat: true}
		if summary != "" {
			notes[text].hold = summary
		}
		return false, summary
	}
	n.count++
	if n.quiet && n.count == 1 {
		return false, ""
	}
	if n.quiet {
		n.quiet = false
		n.unsent = true
		return true, ""
	}
	return false, ""
}

func ack(notes map[string]*note, text string, now time.Time) {
	for _, n := range notes {
		if n.hold == text {
			n.hold = ""
			n.sent = now
			if n.fromRepeat {
				n.quiet = true
				n.count = 0
				n.unsent = false
			}
			return
		}
	}
	n := notes[text]
	if n == nil {
		return
	}
	n.unsent = false
	n.sent = now
}

func stripDial(s string) string {
	return dialHop.ReplaceAllString(s, "dial")
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
