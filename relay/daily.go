package relay

// Daily counts for the operator: how many different hubs used the relay
// each day, and how many of them were new installs. Counts only. A hub is
// remembered for the day as an HMAC of its ID under a key drawn at random
// each day and thrown away at midnight, so the set can't be matched to a
// hub or linked across days. Nothing is written to disk; each finished day
// is one log line and stays in memory for /stats (served on the relay's
// own address only: the public site forwards /v1/ and /healthz alone).
// "New" is the hub's own word (X-OmniLLM-First on its first connection
// ever): the relay keeps no list of hubs it has seen.

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// FirstHeader is sent by a hub on its first connection ever.
const FirstHeader = "X-OmniLLM-First"

// DayCount is one day's numbers.
type DayCount struct {
	Date string `json:"date"` // the relay's local date, 2026-10-24
	Hubs int    `json:"hubs"` // different hubs that connected
	New  int    `json:"new"`  // of them, first connections ever
}

type daily struct {
	mu   sync.Mutex
	now  func() time.Time
	day  string
	key  [32]byte
	seen map[[32]byte]struct{}
	new  int
	past []DayCount // newest last, at most 60
}

func (d *daily) roll(t time.Time, logf func(string, ...any), online func() [][32]byte) {
	date := t.Format("2006-01-02")
	if date == d.day {
		return
	}
	if d.day != "" {
		c := DayCount{Date: d.day, Hubs: len(d.seen), New: d.new}
		d.past = append(d.past, c)
		if len(d.past) > 60 {
			d.past = d.past[len(d.past)-60:]
		}
		logf("relay: daily %s: hubs %d, new %d", c.Date, c.Hubs, c.New)
	}
	d.day, d.new, d.seen = date, 0, map[[32]byte]struct{}{}
	_, _ = rand.Read(d.key[:])
	// Hubs still connected from yesterday count today too.
	if online != nil {
		for _, id := range online() {
			d.seen[d.mark(id)] = struct{}{}
		}
	}
}

func (d *daily) mark(id [32]byte) [32]byte {
	m := hmac.New(sha256.New, d.key[:])
	m.Write(id[:])
	var out [32]byte
	copy(out[:], m.Sum(nil))
	return out
}

// hub counts one hub connection.
func (s *Server) countHub(id [32]byte, first bool) {
	d := &s.daily
	d.mu.Lock()
	defer d.mu.Unlock()
	d.roll(d.clock(), s.logf, nil)
	m := d.mark(id)
	if _, ok := d.seen[m]; !ok {
		d.seen[m] = struct{}{}
		if first {
			d.new++
		}
	}
}

func (d *daily) clock() time.Time {
	if d.now != nil {
		return d.now()
	}
	return time.Now()
}

// RollDay closes the day at midnight even when no hub connects then; the
// relay command calls it every minute.
func (s *Server) RollDay() {
	d := &s.daily
	d.mu.Lock()
	defer d.mu.Unlock()
	d.roll(d.clock(), s.logf, s.onlineIDs)
}

func (s *Server) onlineIDs() [][32]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([][32]byte, 0, len(s.hubs))
	for id := range s.hubs {
		ids = append(ids, id)
	}
	return ids
}

// Stats is today so far and the finished days still in memory.
func (s *Server) Stats() (today DayCount, days []DayCount) {
	s.RollDay()
	d := &s.daily
	d.mu.Lock()
	defer d.mu.Unlock()
	return DayCount{Date: d.day, Hubs: len(d.seen), New: d.new}, append([]DayCount(nil), d.past...)
}

// serveStats answers only on the relay's own machine and only when asked
// directly: a request that came through a proxy, or from another address,
// is refused (a relay listening on the internet itself must not publish it).
func (s *Server) serveStats(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(host)
	if err != nil || ip == nil || !ip.IsLoopback() || r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("X-Real-IP") != "" || r.Header.Get("Forwarded") != "" {
		http.NotFound(w, r)
		return
	}
	today, days := s.Stats()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"today": today, "days": days})
}
