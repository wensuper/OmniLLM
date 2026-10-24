package relay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDailyCountsHubsOncePerDayAndNewOnlyWhenTheHubSaysSo(t *testing.T) {
	var logs []string
	s := New(Options{Open: true, Logf: func(f string, a ...any) { logs = append(logs, fmt.Sprintf(f, a...)) }})
	now := time.Date(2026, 10, 24, 9, 0, 0, 0, time.Local)
	s.daily.now = func() time.Time { return now }
	a, b, c := [32]byte{1}, [32]byte{2}, [32]byte{3}
	s.countHub(a, true)
	s.countHub(a, true) // reconnects: still one hub, one new
	s.countHub(b, false)
	s.countHub(b, false)
	if today, _ := s.Stats(); today.Hubs != 2 || today.New != 1 || today.Date != "2026-10-24" {
		t.Fatalf("today %+v", today)
	}
	markA := s.daily.mark(a)
	now = now.Add(24 * time.Hour)
	s.countHub(c, true)
	today, days := s.Stats()
	if len(days) != 1 || days[0] != (DayCount{Date: "2026-10-24", Hubs: 2, New: 1}) {
		t.Fatalf("yesterday %+v", days)
	}
	if today.Hubs != 1 || today.New != 1 {
		t.Fatalf("today %+v", today)
	}
	if !strings.Contains(strings.Join(logs, "\n"), "relay: daily 2026-10-24: hubs 2, new 1") {
		t.Fatalf("no daily line: %q", logs)
	}
	// A new key each day: yesterday's marks can't be matched to today's.
	if s.daily.mark(a) == markA {
		t.Fatal("the day key didn't change")
	}
	// Only counts, nothing that identifies a hub, in the log line.
	for _, l := range logs {
		if strings.Contains(l, fmt.Sprintf("%x", a[:4])) {
			t.Fatalf("a hub ID in the log: %s", l)
		}
	}
}

func TestStatsOnlyOnTheRelaysOwnMachineAndNotThroughAProxy(t *testing.T) {
	s := New(Options{Open: true})
	get := func(remote string, hdr map[string]string) int {
		r := httptest.NewRequest("GET", "/stats", nil)
		r.RemoteAddr = remote
		for k, v := range hdr {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code == http.StatusOK {
			var v map[string]any
			if json.Unmarshal(w.Body.Bytes(), &v) != nil || v["today"] == nil {
				t.Fatalf("body %s", w.Body)
			}
		}
		return w.Code
	}
	if code := get("127.0.0.1:5000", nil); code != http.StatusOK {
		t.Fatalf("local: %d", code)
	}
	if code := get("203.0.113.7:5000", nil); code == http.StatusOK {
		t.Fatal("served to another address")
	}
	if code := get("127.0.0.1:5000", map[string]string{"X-Forwarded-For": "203.0.113.7"}); code == http.StatusOK {
		t.Fatal("served through a proxy")
	}
}
