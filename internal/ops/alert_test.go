package ops

import (
	"strings"
	"testing"
	"time"
)

func TestStageUnique(t *testing.T) {
	notes := map[string]*note{}
	now := time.Now()
	every := time.Minute

	fresh, summary := stage(notes, "db down", now, every)
	if !fresh || summary != "" {
		t.Fatalf("first: fresh %v summary %q", fresh, summary)
	}
	ack(notes, "db down", now)
	fresh, summary = stage(notes, "db down", now.Add(time.Second), every)
	if fresh || summary != "" || notes["db down"].count != 2 {
		t.Fatalf("repeat: fresh %v summary %q count %d", fresh, summary, notes["db down"].count)
	}
	fresh, _ = stage(notes, "panic", now.Add(time.Second), every)
	if !fresh {
		t.Fatal("other text must send immediately")
	}
	fresh, summary = stage(notes, "db down", now.Add(2*time.Minute), every)
	if !fresh || summary == "" {
		t.Fatalf("after window: fresh %v summary %q", fresh, summary)
	}
	if !strings.Contains(summary, "2") || !strings.Contains(summary, "db down") {
		t.Fatalf("summary %q", summary)
	}
}

func TestRepeatCollapses(t *testing.T) {
	notes := map[string]*note{}
	now := time.Now()
	every := time.Minute
	send, summary := repeat(notes, "getUpdates: eof", now, every)
	if send || summary != "" {
		t.Fatal("single blip stays in the log")
	}
	send, summary = repeat(notes, "getUpdates: eof", now.Add(time.Second), every)
	if !send || summary != "" {
		t.Fatal("second copy must notify once")
	}
	ack(notes, "getUpdates: eof", now.Add(time.Second))
	send, _ = repeat(notes, "getUpdates: eof", now.Add(2*time.Second), every)
	if send || notes["getUpdates: eof"].count != 3 {
		t.Fatalf("later copies only count: send %v count %d", send, notes["getUpdates: eof"].count)
	}
	_, summary = repeat(notes, "getUpdates: eof", now.Add(2*time.Minute), every)
	if !strings.Contains(summary, "3") {
		t.Fatalf("summary %q", summary)
	}
	send, _ = repeat(notes, "other", now, every)
	if send {
		t.Fatal("a different text is its own first blip")
	}
}

func TestDueHoldsRetry(t *testing.T) {
	notes := map[string]*note{}
	now := time.Now()
	stage(notes, "lock lost", now, time.Minute)
	ack(notes, "lock lost", now)
	stage(notes, "lock lost", now.Add(time.Second), time.Minute)
	var jobs []string
	for _, n := range notes {
		if n.count > 1 && now.Add(2*time.Minute).Sub(n.sent) >= time.Minute {
			n.hold = "Повторилась 2 раз:\nlock lost"
			jobs = append(jobs, n.hold)
		}
	}
	if len(jobs) != 1 {
		t.Fatalf("jobs %v", jobs)
	}
	if ackHolds(notes) {
		t.Fatal("hold must stay until acked")
	}
	ack(notes, jobs[0], now.Add(2*time.Minute))
	if notes["lock lost"].hold != "" {
		t.Fatal("hold stuck")
	}
}

func TestRepeatIgnoresDialPort(t *testing.T) {
	a := stripDial("read tcp 10.0.0.5:111->149.154.167.50:443: reset")
	b := stripDial("read tcp 10.0.0.5:222->149.154.167.50:443: reset")
	if a != b {
		t.Fatalf("ports leaked: %q vs %q", a, b)
	}
	notes := map[string]*note{}
	now := time.Now()
	send, _ := repeat(notes, a, now, time.Minute)
	if send {
		t.Fatal("first blip stays in the log")
	}
	send, _ = repeat(notes, b, now.Add(time.Second), time.Minute)
	if !send {
		t.Fatal("second blip with a new port must notify")
	}
}

func TestRepeatAfterSummary(t *testing.T) {
	notes := map[string]*note{
		"getUpdates: eof": {text: "getUpdates: eof", count: 4, fromRepeat: true, sent: time.Now().Add(-2 * time.Minute)},
	}
	now := time.Now()
	_, summary := repeat(notes, "getUpdates: eof", now, time.Minute)
	if !strings.Contains(summary, "4") {
		t.Fatalf("summary %q", summary)
	}
	ack(notes, summary, now)
	send, _ := repeat(notes, "getUpdates: eof", now.Add(time.Second), time.Minute)
	if send {
		t.Fatal("first blip after summary stays in the log")
	}
	send, _ = repeat(notes, "getUpdates: eof", now.Add(2*time.Second), time.Minute)
	if !send {
		t.Fatal("second copy notifies again")
	}
}

func ackHolds(notes map[string]*note) bool {
	for _, n := range notes {
		if n.hold == "" {
			return true
		}
	}
	return false
}
