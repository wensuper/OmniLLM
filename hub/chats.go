package hub

import (
	"context"
	"time"

	"github.com/wensuper/OmniLLM/internal/provider"
	"github.com/wensuper/OmniLLM/store"
)

// chatsKey holds the table of kept sessions: hashes and the programs'
// session ids, no conversation text (design §13.4).
const chatsKey = "chats/index"

func (h *Hub) loadChats(ctx context.Context) {
	h.chats = provider.NewChats(h.opts.Now, func(recs []provider.ChatRecord) {
		if err := store.PutJSON(context.Background(), h.opts.Store, chatsKey, recs); err != nil {
			h.opts.Logf("hub: saving the conversation table: %v", err)
		}
	})
	var recs []provider.ChatRecord
	if err := store.GetJSON(ctx, h.opts.Store, chatsKey, &recs); err == nil {
		h.chats.Load(recs)
	}
}

// ForgetChats deletes every conversation a hub kept in st, and the table.
// For uninstalling, with the hub stopped.
func ForgetChats(ctx context.Context, st store.Store) {
	var recs []provider.ChatRecord
	if err := store.GetJSON(ctx, st, chatsKey, &recs); err == nil {
		provider.RemoveChats(recs)
	} else {
		provider.RemoveChats(nil)
	}
	_ = st.Delete(ctx, chatsKey)
}

// sweepChats removes conversations unused for a day every hour until the
// hub stops (Start sweeps once itself).
func (h *Hub) sweepChats(ctx context.Context) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.chats.Sweep()
		}
	}
}
