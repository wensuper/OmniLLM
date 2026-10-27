package hub

import (
	"fmt"

	omnillm "github.com/wensuper/OmniLLM"
)

// A model that can't read images (a text-only local model) is not made to
// fail a chat over pictures from earlier turns: they are left out and the
// model is told one was there (design §5.4). A picture in the new message
// still gets MsgNoVision, since answering it unseen would mislead.

// newMessageHasImages: the message being answered carries a picture.
func newMessageHasImages(req omnillm.Request) bool {
	n := len(req.Messages)
	return n > 0 && len(req.Messages[n-1].Images) > 0
}

// withoutImages is req with every picture replaced by a short note.
func withoutImages(req omnillm.Request) omnillm.Request {
	msgs := make([]omnillm.Message, len(req.Messages))
	for i, m := range req.Messages {
		if n := len(m.Images); n > 0 {
			note := fmt.Sprintf("[%d image(s) left out: this model can't read images]", n)
			if m.Content != "" {
				note = m.Content + "\n" + note
			}
			m.Content, m.Images = note, nil
		}
		msgs[i] = m
	}
	req.Messages = msgs
	return req
}

// knownNoVision: the model refused pictures before (since the hub started).
func (h *Hub) knownNoVision(model string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.noVision[model]
}

func (h *Hub) markNoVision(model string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.noVision == nil {
		h.noVision = map[string]bool{}
	}
	h.noVision[model] = true
}
