//go:build js && wasm

// Command omniweb is the device side of OmniLLM for a web browser (phones
// and tablets): the same device package as `omnillm connect`, compiled to
// WebAssembly. It exposes a small API on globalThis.omnillm for the web
// client page; the page stores the pairing state (encrypted with a key the
// browser cannot export) and hands it back on each start.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"sync"
	"syscall/js"
	"time"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/secret"
)

var (
	mu      sync.Mutex
	dev     *device.Device
	cancels = map[string]context.CancelFunc{}
)

func main() {
	api := map[string]any{
		"pair":   js.FuncOf(pair),
		"open":   js.FuncOf(open),
		"models": js.FuncOf(models),
		"chat":   js.FuncOf(chat),
		"cancel": js.FuncOf(cancel),
		"close":  js.FuncOf(closeDev),
		"check":  js.FuncOf(check),
	}
	js.Global().Set("omnillm", js.ValueOf(api))
	js.Global().Call("dispatchEvent", js.Global().Get("Event").New("omnillm-ready"))
	select {}
}

// promise runs f in a goroutine and settles a JS promise with its result.
func promise(f func() (any, error)) js.Value {
	return js.Global().Get("Promise").New(js.FuncOf(func(_ js.Value, a []js.Value) any {
		resolve, reject := a[0], a[1]
		go func() {
			v, err := f()
			if err != nil {
				reject.Invoke(errValue(err))
				return
			}
			resolve.Invoke(js.ValueOf(v))
		}()
		return nil
	}))
}

// errValue gives the page a code and a sentence in its language; never a
// raw error from deep inside.
func errValue(err error) js.Value {
	lang := js.Global().Get("navigator").Get("language").String()
	l := "en"
	if len(lang) >= 2 && lang[:2] == "zh" {
		l = "zh"
	}
	code, msg := "error", ""
	var oe *omnillm.Error
	if errors.As(err, &oe) {
		code, msg = string(oe.Code), oe.Msg
	} else if errors.Is(err, context.Canceled) {
		code = "cancelled"
	}
	return js.ValueOf(map[string]any{"code": code, "msg": msg, "message": omnillm.Describe(l, err)})
}

// state is the pairing as the page stores it (values base64).
func exportState(s *secret.Memory) (string, error) {
	ctx := context.Background()
	names, err := s.List(ctx, "")
	if err != nil {
		return "", err
	}
	sort.Strings(names)
	out := map[string]string{}
	for _, n := range names {
		v, err := s.Get(ctx, n)
		if err != nil {
			return "", err
		}
		out[n] = base64.StdEncoding.EncodeToString(v)
	}
	b, err := json.Marshal(out)
	return string(b), err
}

func importState(st string) (*secret.Memory, error) {
	var in map[string]string
	if err := json.Unmarshal([]byte(st), &in); err != nil {
		return nil, errors.New("the saved pairing is damaged")
	}
	s := secret.NewMemory()
	for n, v := range in {
		b, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, errors.New("the saved pairing is damaged")
		}
		if err := s.Set(context.Background(), n, b); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// check(text) validates a pairing text without using it.
func check(_ js.Value, a []js.Value) any {
	return promise(func() (any, error) {
		return nil, device.CheckPairingText(a[0].String())
	})
}

// pair(text, name, onCode) → state JSON.
func pair(_ js.Value, a []js.Value) any {
	text, name, onCode := a[0].String(), a[1].String(), a[2]
	return promise(func() (any, error) {
		s := secret.NewMemory()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		_, err := device.Pair(ctx, device.Options{Secrets: s}, text, name, func(code string) { onCode.Invoke(code) })
		if err != nil {
			return nil, err
		}
		return exportState(s)
	})
}

// open(state) connects with a saved pairing.
func open(_ js.Value, a []js.Value) any {
	st := a[0].String()
	return promise(func() (any, error) {
		s, err := importState(st)
		if err != nil {
			return nil, err
		}
		d, err := device.Open(context.Background(), device.Options{Secrets: s, AckTimeout: 8 * time.Second})
		if err != nil {
			return nil, err
		}
		mu.Lock()
		if dev != nil {
			dev.Close()
		}
		dev = d
		mu.Unlock()
		return nil, nil
	})
}

func current() (*device.Device, error) {
	mu.Lock()
	defer mu.Unlock()
	if dev == nil {
		return nil, errors.New("not paired")
	}
	return dev, nil
}

func models(_ js.Value, _ []js.Value) any {
	return promise(func() (any, error) {
		d, err := current()
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ms, err := d.Models(ctx)
		if err != nil {
			return nil, err
		}
		out := []any{}
		for _, m := range ms {
			out = append(out, map[string]any{"id": m.ID, "name": m.DisplayName, "provider": m.Provider, "humanOnly": m.HumanOnly})
		}
		return out, nil
	})
}

// chat(id, model, messagesJSON, onDelta) → {in, out, finish}.
func chat(_ js.Value, a []js.Value) any {
	id, model, msgsJSON, onDelta := a[0].String(), a[1].String(), a[2].String(), a[3]
	return promise(func() (any, error) {
		d, err := current()
		if err != nil {
			return nil, err
		}
		var msgs []omnillm.Message
		if err := json.Unmarshal([]byte(msgsJSON), &msgs); err != nil {
			return nil, err
		}
		ctx, cancelFn := context.WithTimeout(context.Background(), 15*time.Minute)
		mu.Lock()
		cancels[id] = cancelFn
		mu.Unlock()
		defer func() {
			mu.Lock()
			delete(cancels, id)
			mu.Unlock()
			cancelFn()
		}()
		// A person typed this: the hub lets subscription models answer.
		req := omnillm.Request{Model: model, Messages: msgs, Caller: omnillm.Interactive}
		for retried := false; ; retried = true {
			s, err := d.Complete(ctx, req)
			if err != nil {
				return nil, err
			}
			got := false
			for {
				c, err := s.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				// The connection dropped before any of the reply came (a
				// phone switching networks with a picture on its way): the
				// hub has stopped that call, so ask once more.
				if err != nil && !got && !retried && errors.Is(err, omnillm.ErrHubOffline) && ctx.Err() == nil {
					s.Close()
					s = nil
					break
				}
				if err != nil {
					s.Close()
					return nil, err
				}
				if c.Model != "" {
					onDelta.Invoke("", c.Model) // which model and version is answering
				}
				if c.Text != "" {
					got = true
					onDelta.Invoke(c.Text)
				}
			}
			if s == nil {
				continue
			}
			u := s.Usage()
			s.Close()
			return map[string]any{"in": u.InputTokens, "out": u.OutputTokens, "finish": s.FinishReason()}, nil
		}
	})
}

func cancel(_ js.Value, a []js.Value) any {
	mu.Lock()
	if c := cancels[a[0].String()]; c != nil {
		c()
	}
	mu.Unlock()
	return nil
}

func closeDev(_ js.Value, _ []js.Value) any {
	mu.Lock()
	if dev != nil {
		dev.Close()
		dev = nil
	}
	mu.Unlock()
	return nil
}
