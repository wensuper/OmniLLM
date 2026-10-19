package hub

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/internal/e2e"
	"github.com/wensuper/OmniLLM/internal/proto"
	"github.com/wensuper/OmniLLM/internal/relayproto"
	"github.com/wensuper/OmniLLM/store"
)

// A removed device that was offline at the time only ever saw "can't reach
// the hub" before: through the relay, a removed mailbox looks exactly like
// a hub that is asleep. So for a while the hub keeps the removed device's
// mailbox and key (a tombstone). When that device comes back it proves its
// key as usual and is told, inside the encrypted session, that it was
// removed; it can then forget its pairing. Nothing else is served, and
// the session ends after a short while.
const tombstoneFor = 30 * 24 * time.Hour

type tombstone struct {
	Device  Device    `json:"device"`
	Removed time.Time `json:"removed"`
}

func (h *Hub) putTombstone(ctx context.Context, d Device) error {
	return store.PutJSON(ctx, h.opts.Store, "revoked/"+d.ID, tombstone{Device: d, Removed: h.opts.Now().UTC()})
}

// tombstones lists the live tombstones and drops expired ones.
func (h *Hub) tombstones(ctx context.Context) []tombstone {
	keys, _ := h.opts.Store.List(ctx, "revoked/")
	var out []tombstone
	for _, k := range keys {
		var t tombstone
		if store.GetJSON(ctx, h.opts.Store, k, &t) != nil {
			continue
		}
		if h.opts.Now().Sub(t.Removed) > tombstoneFor {
			_ = h.opts.Store.Delete(ctx, k)
			continue
		}
		out = append(out, t)
	}
	return out
}

func (h *Hub) tombstoneByMailbox(ctx context.Context, id relayproto.ID) (tombstone, bool) {
	for _, t := range h.tombstones(ctx) {
		if t.Device.Mailbox == id.String() {
			return t, true
		}
	}
	return tombstone{}, false
}

// serveRemoved answers a removed device: after it proves its key, every
// request it sends gets "this device was removed", then the session ends.
func (h *Hub) serveRemoved(ctx context.Context, c *mailboxConn, t tombstone) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	sess, err := e2e.Accept(ctx, c, c.id[:], h.key, nil, func(peer []byte) error {
		if subtle.ConstantTimeCompare(peer, t.Device.PublicKey) != 1 {
			return errors.New("hub: unknown device key")
		}
		return nil
	})
	if err != nil {
		return
	}
	defer sess.Close()
	for i := 0; i < 20; i++ {
		b, err := sess.Recv(ctx)
		if err != nil {
			return
		}
		env, err := proto.Unmarshal(b)
		if err != nil {
			return
		}
		if env.ID != 0 {
			h.sendErr(ctx, sess, env.ID, &omnillm.Error{Code: omnillm.CodeInvalidToken, Msg: omnillm.MsgDeviceRevoked})
		}
	}
}
