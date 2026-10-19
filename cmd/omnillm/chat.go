package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/secret"
	"golang.org/x/term"
)

func openDevice(ctx context.Context, sec secret.Store) *device.Device {
	d, err := device.Open(ctx, device.Options{Secrets: sec})
	if err != nil {
		die(errors.New(t("这台设备还没配对，先运行 omnillm connect pair。", "This device is not paired yet. Run omnillm connect pair first.")))
	}
	return d
}

// atTerminal: a person is typing here. Only then does a call say a person
// is waiting (Interactive); from a script, cron or a pipe it is Background,
// and subscription models, kept for people, are not offered.
func atTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func callerHere() omnillm.Caller {
	if atTerminal() {
		return omnillm.Interactive
	}
	return omnillm.Background
}

// defaultModel picks what a person most likely wants: the Claude
// subscription's Sonnet, else any subscription model, else the first.
func defaultModel(ctx context.Context, d *device.Device) string {
	ms, err := d.Models(ctx)
	if err != nil {
		die(err)
	}
	return pickModel(ms, atTerminal())
}

func pickModel(ms []omnillm.ModelInfo, person bool) string {
	if person {
		for _, m := range ms {
			if m.ID == "claude-code/sonnet" {
				return m.ID
			}
		}
	}
	for _, m := range ms {
		if m.HumanOnly == person {
			return m.ID
		}
	}
	if !person {
		die(errors.New(t("在脚本里（没有终端）只能用 API Key 或本地模型；主机上只有订阅模型，它们只给人在终端里对话用。",
			"From a script (no terminal) only API-key or local models can be used; the hub has only subscription models, which are for a person typing in a terminal.")))
	}
	if len(ms) == 0 {
		die(errors.New(t("主机上还没有可用的模型。", "The hub has no models yet.")))
	}
	return ms[0].ID
}

// chatLoop is a conversation in the terminal. Ctrl-C stops a reply; Ctrl-D
// or /exit leaves. /new starts over, /model switches models.
func chatLoop(ctx context.Context, d *device.Device, model string) {
	ms, err := d.Models(ctx)
	if err != nil {
		die(err)
	}
	if len(ms) == 0 {
		die(errors.New(t("主机上还没有可用的模型。", "The hub has no models yet.")))
	}
	if model == "" {
		model = pickModel(ms, atTerminal())
	}
	name := func(id string) string {
		for _, m := range ms {
			if m.ID == id {
				return m.DisplayName
			}
		}
		return id
	}
	fmt.Fprintf(os.Stderr, t("和 %s 对话。/model 换模型，/new 重新开始，Ctrl-C 停止回答，Ctrl-D 退出。\n",
		"Chatting with %s. /model to switch, /new to start over, Ctrl-C stops a reply, Ctrl-D quits.\n"), name(model))
	in := bufio.NewReader(os.Stdin)
	var msgs []omnillm.Message
	for {
		fmt.Fprint(os.Stderr, "\n> ")
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			fmt.Fprintln(os.Stderr)
			return
		}
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			continue
		case line == "/exit" || line == "/quit":
			return
		case line == "/new":
			msgs = nil
			fmt.Fprintln(os.Stderr, t("已重新开始。", "Started over."))
			continue
		case line == "/model" || strings.HasPrefix(line, "/model "):
			arg := strings.TrimSpace(strings.TrimPrefix(line, "/model"))
			if arg == "" {
				for i, m := range ms {
					mark := " "
					if m.ID == model {
						mark = "*"
					}
					fmt.Fprintf(os.Stderr, "%s %d. %s  (%s)\n", mark, i+1, m.DisplayName, m.ID)
				}
				fmt.Fprintln(os.Stderr, t("输入 /model 序号 或 /model ID 切换。", "Type /model NUMBER or /model ID to switch."))
				continue
			}
			found := ""
			for i, m := range ms {
				if arg == m.ID || arg == fmt.Sprint(i+1) {
					found = m.ID
				}
			}
			if found == "" {
				fmt.Fprintln(os.Stderr, t("没有这个模型。", "No such model."))
				continue
			}
			model = found
			fmt.Fprintf(os.Stderr, t("现在用 %s。\n", "Now using %s.\n"), name(model))
			continue
		}
		msgs = append(msgs, omnillm.Message{Role: "user", Content: line})
		reply, err := chatOnce(ctx, d, model, msgs)
		if reply != "" {
			msgs = append(msgs, omnillm.Message{Role: "assistant", Content: reply})
		} else {
			msgs = msgs[:len(msgs)-1]
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, omnillm.Describe(lang(), err))
		}
		if ctx.Err() != nil {
			return
		}
	}
}

// chatOnce streams one reply; Ctrl-C stops it without leaving the chat.
func chatOnce(ctx context.Context, d *device.Device, model string, msgs []omnillm.Message) (string, error) {
	rctx, cancel := context.WithCancel(ctx)
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer signal.Stop(sig)
	go func() {
		select {
		case <-sig:
			cancel()
		case <-rctx.Done():
		}
	}()
	s, err := d.Complete(rctx, omnillm.Request{Model: model, Caller: callerHere(), Messages: msgs})
	if err != nil {
		return "", err
	}
	defer s.Close()
	var b strings.Builder
	for {
		c, err := s.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println()
			return b.String(), nil
		}
		if err != nil {
			fmt.Println()
			if rctx.Err() != nil && ctx.Err() == nil {
				fmt.Fprintln(os.Stderr, t("（已停止）", "(stopped)"))
				return b.String(), nil
			}
			return b.String(), err
		}
		b.WriteString(c.Text)
		fmt.Print(c.Text)
	}
}
