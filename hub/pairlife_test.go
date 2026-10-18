package hub_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/hub"
	"github.com/wensuper/OmniLLM/secret"
)

// A pairing whose owner command went away must free its slot at once;
// before, eight abandoned pairings blocked new ones for five minutes.
func TestAbandonedPairingsFreeTheirSlots(t *testing.T) {
	e := setup(t, nil)
	confirm := func(context.Context, hub.PairRequest) (bool, error) { return true, nil }
	for i := 0; i < 12; i++ {
		p, err := e.hub.Pair(context.Background(), hub.PairOptions{Confirm: confirm})
		if err != nil {
			t.Fatalf("pairing %d: %v", i, err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := p.Wait(ctx); err == nil {
			t.Fatal("Wait with a dead context should fail")
		}
	}
}

func TestCancelledPairingCodeIsDead(t *testing.T) {
	e := setup(t, nil)
	var asked atomic.Bool
	p, err := e.hub.Pair(context.Background(), hub.PairOptions{Confirm: func(context.Context, hub.PairRequest) (bool, error) {
		asked.Store(true)
		return true, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	p.Cancel()
	p.Cancel() // twice is fine
	time.Sleep(200 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := device.Pair(ctx, device.Options{Secrets: secret.NewMemory()}, p.Code, "late", nil); err == nil || asked.Load() {
		t.Fatalf("cancelled code worked: %v asked=%v", err, asked.Load())
	}
}

// Cancelling while the owner is being asked ends the question.
func TestCancelEndsOwnerQuestion(t *testing.T) {
	e := setup(t, nil)
	inQuestion := make(chan struct{})
	questionDone := make(chan error, 1)
	p, err := e.hub.Pair(context.Background(), hub.PairOptions{Confirm: func(ctx context.Context, _ hub.PairRequest) (bool, error) {
		close(inQuestion)
		<-ctx.Done()
		questionDone <- ctx.Err()
		return false, ctx.Err()
	}})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = device.Pair(ctx, device.Options{Secrets: secret.NewMemory()}, p.Code, "phone", nil)
	}()
	select {
	case <-inQuestion:
	case <-time.After(10 * time.Second):
		t.Fatal("owner was never asked")
	}
	p.Cancel()
	select {
	case <-questionDone:
	case <-time.After(5 * time.Second):
		t.Fatal("the owner question outlived the cancelled pairing")
	}
}
