package hub_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/secret"
)

// The console's Usage tab: per model as well as per device, and over a
// range of days.
func TestUsagePerModelAndDayRange(t *testing.T) {
	var offset atomic.Int64 // days added to the real clock
	now := func() time.Time { return time.Now().Add(time.Duration(offset.Load()) * 24 * time.Hour) }
	e := setup(t, func(o *hub.Options) { o.Now = now })
	ctx := context.Background()
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatal(err)
	}
	offset.Store(1) // the next day
	for _, m := range []string{"fake/m1", "fake/m2"} {
		if _, _, err := ask(ctx, e.hub, m, "hello", omnillm.Interactive); err != nil {
			t.Fatal(err)
		}
	}
	calls := func(rows []hub.Spend, who string) int64 {
		for _, r := range rows {
			if r.Who == who {
				return r.Calls
			}
		}
		return 0
	}
	today, _ := e.hub.Usage(ctx, "")
	both, err := e.hub.UsageBetween(ctx, e.hub.Day(1), e.hub.Day(0))
	if err != nil {
		t.Fatal(err)
	}
	if calls(today, "host") != 2 || calls(both, "host") != 3 || calls(both, "_total") != 3 {
		t.Fatalf("day range: today %+v both %+v", today, both)
	}
	yesterday, _ := e.hub.ModelUsageBetween(ctx, e.hub.Day(1), e.hub.Day(1))
	week, _ := e.hub.ModelUsageBetween(ctx, e.hub.Day(6), e.hub.Day(0))
	if calls(yesterday, "fake/m1") != 1 || calls(yesterday, "fake/m2") != 0 ||
		calls(week, "fake/m1") != 2 || calls(week, "fake/m2") != 1 {
		t.Fatalf("per model: yesterday %+v week %+v", yesterday, week)
	}
	for _, r := range week {
		if r.Name == "" || r.Removed {
			t.Fatalf("a model line without its name: %+v", r)
		}
	}
	if _, err := e.hub.UsageBetween(ctx, e.hub.Day(0), e.hub.Day(1)); err == nil {
		t.Fatal("a range that ends before it starts was accepted")
	}
}

// A device with a session open is Online; once it closes, it is not.
func TestDeviceOnlineFlag(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	d, _ := e.pairOK()
	online := func() bool {
		devs, err := e.hub.Devices(ctx)
		if err != nil || len(devs) != 1 {
			t.Fatalf("devices %v %v", devs, err)
		}
		return devs[0].Online
	}
	if _, err := d.Models(ctx); err != nil {
		t.Fatal(err)
	}
	if !online() {
		t.Fatal("a device in a session is not online")
	}
	d.Close()
	deadline := time.Now().Add(5 * time.Second)
	for online() {
		if time.Now().After(deadline) {
			t.Fatal("a closed device stays online")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// The device side going away while the owner is asked is reported as
// such, not as the owner's refusal.
func TestPairingDeviceLeftIsNotDenied(t *testing.T) {
	e := setup(t, nil)
	inQuestion := make(chan struct{})
	p, err := e.hub.Pair(context.Background(), hub.PairOptions{Confirm: func(ctx context.Context, _ hub.PairRequest) (bool, error) {
		close(inQuestion)
		<-ctx.Done()
		return false, ctx.Err()
	}})
	if err != nil {
		t.Fatal(err)
	}
	dctx, dcancel := context.WithCancel(context.Background())
	go func() { _, _ = device.Pair(dctx, device.Options{Secrets: secret.NewMemory()}, p.Code, "phone", nil) }()
	select {
	case <-inQuestion:
	case <-time.After(10 * time.Second):
		t.Fatal("owner was never asked")
	}
	dcancel() // the phone cancels
	wctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = p.Wait(wctx)
	if !errors.Is(err, hub.ErrPairingDeviceLeft) || errors.Is(err, omnillm.ErrPairingDenied) {
		t.Fatalf("want the device-left reason, got %v", err)
	}
}

// The owner may name the device while confirming; a taken name still
// gets a number.
func TestConfirmNamesTheDevice(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	pairAs := func(name string) device.Info {
		t.Helper()
		pctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		p, err := e.hub.Pair(pctx, hub.PairOptions{ConfirmName: func(context.Context, hub.PairRequest) (bool, string, error) {
			return true, name, nil
		}})
		if err != nil {
			t.Fatal(err)
		}
		info, err := device.Pair(pctx, device.Options{Secrets: secret.NewMemory()}, p.Code, "iPhone", nil)
		if err != nil {
			t.Fatal(err)
		}
		return info
	}
	pairAs("Kitchen iPad")
	pairAs("Kitchen iPad")
	pairAs("") // keeps the device's own name
	devs, _ := e.hub.Devices(ctx)
	var names []string
	for _, d := range devs {
		names = append(names, d.Name)
	}
	if len(names) != 3 || names[0] != "Kitchen iPad" || names[1] != "Kitchen iPad (2)" || names[2] != "iPhone" {
		t.Fatalf("names %q", names)
	}
}

func TestPauseAndResume(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	d, _ := e.pairOK()
	devs, _ := e.hub.Devices(ctx)
	id := devs[0].ID
	if _, err := e.hub.UpdateDevice(ctx, id, nil, nil, ptr(true), nil); err != nil {
		t.Fatal(err)
	}
	before := e.fake.calls.Load()
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrDevicePaused) {
		t.Fatalf("a paused device was served: %v", err)
	}
	if e.fake.calls.Load() != before {
		t.Fatal("a paused device's call reached the provider")
	}
	if devs, _ := e.hub.Devices(ctx); !devs[0].Paused {
		t.Fatal("pause not kept")
	}
	if _, err := e.hub.UpdateDevice(ctx, id, nil, nil, ptr(false), nil); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatalf("a resumed device: %v", err)
	}

	a, tok, err := e.hub.AddApp(ctx, "script", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.hub.UpdateApp(ctx, a.ID, nil, nil, ptr(true), nil); err != nil {
		t.Fatal(err)
	}
	req := omnillm.Request{Model: "fake/m1", Messages: []omnillm.Message{{Role: "user", Content: "hello"}}}
	if _, err := e.hub.CompleteAsApp(ctx, tok, req); !errors.Is(err, omnillm.ErrDevicePaused) || omnillm.AsError(err).Msg != omnillm.MsgAppPaused {
		t.Fatalf("a paused app was served: %v", err)
	}
	if _, err := e.hub.UpdateApp(ctx, a.ID, nil, nil, ptr(false), nil); err != nil {
		t.Fatal(err)
	}
	s, err := e.hub.CompleteAsApp(ctx, tok, req)
	if err != nil {
		t.Fatalf("a resumed app: %v", err)
	}
	s.Close()
}

// A model the owner hides is not for devices any more; a rename shows.
func TestHiddenAndRenamedModels(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	d, _ := e.pairOK()
	if err := e.hub.SetModelPref(ctx, "fake/m1", ptr("My model"), ptr(false)); err != nil {
		t.Fatal(err)
	}
	ms, _ := e.hub.OwnerModels(ctx)
	found := false
	for _, m := range ms {
		if m.ID == "fake/m1" {
			found = true
			if !m.Hidden || m.DisplayName != "My model" {
				t.Fatalf("owner view %+v", m)
			}
		}
	}
	if !found {
		t.Fatal("a hidden model left the owner's own list")
	}
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrModelNotAllowed) {
		t.Fatalf("a device used a hidden model: %v", err)
	}
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatalf("the host lost a model hidden from devices: %v", err)
	}
	if err := e.hub.SetModelPref(ctx, "fake/m1", ptr(""), ptr(true)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatalf("shown again: %v", err)
	}
}

// Something left today, but not enough for this call's worst case: a
// different sentence from "used up".
func TestBudgetWouldExceed(t *testing.T) {
	e := setup(t, func(o *hub.Options) { o.HostBudget = 310_000; o.GlobalBudget = 1_000_000_000 })
	ctx := context.Background()
	if _, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive); err != nil {
		t.Fatal(err)
	}
	_, _, err := ask(ctx, e.hub, "fake/m1", "hello", omnillm.Interactive)
	if oe := omnillm.AsError(err); err == nil || oe.Code != omnillm.CodeBudgetExceeded || oe.Msg != omnillm.MsgBudgetWouldExceed {
		t.Fatalf("want would-exceed, got %v", err)
	}
}

// A device limited to some models is refused the others and lists only
// its own; apps never get chat-only models; unknown ids are refused.
func TestRestrictDeviceAndAppModels(t *testing.T) {
	e := setup(t, nil)
	ctx := context.Background()
	d, _ := e.pairOK()
	devs, _ := e.hub.Devices(ctx)
	id := devs[0].ID
	if _, err := e.hub.UpdateDevice(ctx, id, nil, nil, nil, ptr([]string{"fake/m2"})); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ask(ctx, d, "fake/m1", "hello", omnillm.Interactive); !errors.Is(err, omnillm.ErrModelNotAllowed) {
		t.Fatalf("a restricted device used another model: %v", err)
	}
	if _, _, err := ask(ctx, d, "fake/m2", "hello", omnillm.Interactive); err != nil {
		t.Fatalf("its own model: %v", err)
	}
	if ms, err := d.Models(ctx); err != nil || len(ms) != 1 || ms[0].ID != "fake/m2" {
		t.Fatalf("restricted list %v %v", ms, err)
	}
	if _, err := e.hub.UpdateDevice(ctx, id, nil, nil, nil, ptr([]string{"fake/nope"})); err == nil {
		t.Fatal("an unknown model id was accepted")
	}
	if _, err := e.hub.UpdateDevice(ctx, id, nil, nil, nil, ptr([]string{})); err != nil {
		t.Fatal(err)
	}
	if ms, _ := d.Models(ctx); len(ms) != 2 {
		t.Fatalf("an empty list is all models: %v", ms)
	}
	a, _, err := e.hub.AddApp(ctx, "script", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.hub.SetModel(ctx, "fake/m1", hubModelHumanOnly()); err != nil {
		t.Fatal(err)
	}
	if _, err := e.hub.UpdateApp(ctx, a.ID, nil, nil, nil, ptr([]string{"fake/m1"})); !errors.Is(err, omnillm.ErrModelNotAllowed) {
		t.Fatalf("an app got a chat-only model: %v", err)
	}
	if got, err := e.hub.UpdateApp(ctx, a.ID, nil, nil, nil, ptr([]string{"fake/m2"})); err != nil || len(got.Models) != 1 {
		t.Fatalf("app models %v %v", got.Models, err)
	}
}
