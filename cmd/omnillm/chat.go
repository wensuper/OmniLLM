package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	omnillm "github.com/wensuper/OmniLLM"
	"github.com/wensuper/OmniLLM/device"
	"github.com/wensuper/OmniLLM/secret"
	"golang.org/x/term"
)

// openFailed says why this device can't be used: not paired (and where it
// looked, since another user or data folder is the usual cause), or the
// saved pairing can't be read.
func openFailed(ctx context.Context, sec secret.Store, dir string, err error) {
	paired, cerr := device.CheckPaired(ctx, sec)
	switch {
	case cerr != nil:
		die(sayf(t("读不出这台设备保存的配对（%s）。请用配对时的同一个用户运行。", "Can’t read this device’s saved pairing (%s). Run as the same user it was paired as."), dir))
	case !paired && role() == "hub":
		die(say(runnable(t("这台电脑就是主机；omnillm connect 命令在和它配对的电脑上用。想在这里试模型，用 omnillm open 打开控制台。", "This computer is the hub; omnillm connect commands run on a computer paired with it. To try the models here, open the console with omnillm open."))))
	case !paired:
		die(sayf(t("这台设备还没配对（在 %s 里没有找到配对），先运行 omnillm connect pair。", "This device isn’t paired yet (no pairing in %s). Run omnillm connect pair first."), dir))
	}
	die(err)
}

func openDevice(ctx context.Context, sec secret.Store, dir string) *device.Device {
	d, err := device.Open(ctx, device.Options{Secrets: sec})
	if err != nil {
		openFailed(ctx, sec, dir, err)
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
	deviceHubOS = d.HubOS()
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
	if len(ms) == 0 {
		die(say(noModelsHere()))
	}
	if !person {
		die(say(omnillm.ForHub(lang(), deviceHubOS, t("输出接到管道或文件、或从脚本运行时，不能用只能手动对话的模型；这台设备能用的模型现在都只能在终端里手动对话。",
			"With output going to a pipe or file, or from a script, chat-only models can't be used, and every model this device can use is chat-only (for chatting in a terminal)."))))
	}
	if len(ms) == 0 {
		die(say(noModelsHere()))
	}
	return ms[0].ID
}

// chatLoop is a conversation in the terminal. Ctrl-C stops a reply; Ctrl-D
// or /exit leaves. /new starts over, /model switches models.
func chatLoop(ctx context.Context, d *device.Device, model string) {
	ms, err := d.Models(ctx)
	deviceHubOS = d.HubOS()
	if err != nil {
		die(err)
	}
	if len(ms) == 0 {
		die(say(noModelsHere()))
	}
	if model == "" {
		model = pickModel(ms, atTerminal())
	} else if !slices.ContainsFunc(ms, func(m omnillm.ModelInfo) bool { return m.ID == model }) {
		die(say(t("没有这个模型。运行 omnillm connect models 查看。", "No such model. Run omnillm connect models to see them.")))
	}
	name := func(id string) string {
		for _, m := range ms {
			if m.ID == id {
				return modelName(m)
			}
		}
		return id
	}
	fmt.Fprintf(os.Stderr, t("和 %s 对话。/model 换模型，/image 文件 附一张图片，/new 重新开始，Ctrl-C 停止回答，Ctrl-D 退出。\n",
		"Chatting with %s. /model to switch, /image FILE to attach a picture, /new to start over, Ctrl-C stops a reply, Ctrl-D quits.\n"), name(model))
	in := bufio.NewReader(os.Stdin)
	var msgs []omnillm.Message
	var pending []omnillm.Image // attached with /image, sent with the next message
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
			msgs, pending = nil, nil
			fmt.Fprintln(os.Stderr, t("已重新开始。", "Started over."))
			continue
		case line == "/image" || strings.HasPrefix(line, "/image "):
			path := strings.TrimSpace(strings.TrimPrefix(line, "/image"))
			if path == "" {
				fmt.Fprintln(os.Stderr, t("用法：/image 文件", "Usage: /image FILE"))
				continue
			}
			im, err := readImage(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, describe(err))
				continue
			}
			if len(pending) >= omnillm.MaxImagesPerMsg {
				fmt.Fprintln(os.Stderr, describe(&omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgBadImage}))
				continue
			}
			pending = append(pending, im)
			fmt.Fprintf(os.Stderr, t("已附上图片（%d 张），和下一条消息一起发。\n", "Image attached (%d); it goes with your next message.\n"), len(pending))
			continue
		case line == "/model" || strings.HasPrefix(line, "/model "):
			arg := strings.TrimSpace(strings.TrimPrefix(line, "/model"))
			if arg == "" {
				for i, m := range ms {
					mark := " "
					if m.ID == model {
						mark = "*"
					}
					fmt.Fprintf(os.Stderr, "%s %d. %s  (%s)\n", mark, i+1, modelName(m), m.ID)
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
				fmt.Fprintln(os.Stderr, t("没有这个模型。输入 /model 看列表。", "No such model. Type /model to see the list."))
				continue
			}
			model = found
			fmt.Fprintf(os.Stderr, t("现在用 %s。\n", "Now using %s.\n"), name(model))
			continue
		}
		// A lone /word that isn't a command is a typo, not a message.
		if w := strings.Fields(line); len(w) == 1 && strings.HasPrefix(line, "/") && !strings.Contains(line[1:], "/") {
			fmt.Fprintln(os.Stderr, t("没有这个命令。可以用 /model、/image 文件、/new、/exit。", "No such command. Use /model, /image FILE, /new or /exit."))
			continue
		}
		msgs = append(msgs, omnillm.Message{Role: "user", Content: line, Images: pending})
		pending = nil
		reply, err := chatOnce(ctx, d, model, msgs)
		if reply != "" {
			msgs = append(msgs, omnillm.Message{Role: "assistant", Content: reply})
		} else {
			msgs = msgs[:len(msgs)-1]
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, describe(err))
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
		if rctx.Err() != nil && ctx.Err() == nil { // Ctrl-C before the first word
			fmt.Fprintln(os.Stderr, t("（已停止）", "(stopped)"))
			return "", nil
		}
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
		fmt.Print(termSafe(c.Text))
	}
}

// readImage loads a picture file for a message (PNG, JPEG, GIF or WebP up
// to 2 MB; the hub checks it again).
func readImage(path string) (omnillm.Image, error) {
	path = strings.Trim(path, "'\"")
	path = strings.ReplaceAll(path, "\\ ", " ") // a dragged-in path: "My\ Pic.png"
	if strings.HasPrefix(path, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(h, path[2:])
		}
	}
	st, err := os.Stat(path)
	if err != nil {
		return omnillm.Image{}, sayf(t("读不了图片文件 %s。", "Can’t read the image file %s."), path)
	}
	if st.Size() > omnillm.MaxImageBytes {
		return omnillm.Image{}, &omnillm.Error{Code: omnillm.CodeBadRequest, Msg: omnillm.MsgBadImage}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return omnillm.Image{}, sayf(t("读不了图片文件 %s。", "Can’t read the image file %s."), path)
	}
	return omnillm.Image{MIME: http.DetectContentType(b), Data: b}, nil
}

// termSafe is model output as a terminal shows it: no control characters
// but line breaks and tabs, so a reply can't move the cursor, fake a
// prompt, set links or write the clipboard (escape sequences). Output to a
// pipe or file is left as it is.
func termSafe(s string) string {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return s
	}
	return stripControls(s)
}

// stripControls drops control characters except line breaks and tabs.
func stripControls(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		// Bidi controls too: they would reorder what the terminal shows.
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) || unicode.Is(unicode.Bidi_Control, r) {
			return -1
		}
		return r
	}, s)
}

// noModelsHere: this device can use no model now (none connected on the
// other computer, or none of the ones it may use is on).
func noModelsHere() string {
	return omnillm.ForHub(lang(), deviceHubOS, t("这台设备现在没有能用的模型。在你的 Mac 上接入一个模型，或用 omnillm hub devices 看它能用哪些模型（omnillm hub device set 名字 --all-models 放开）。",
		"No models are available to this device right now. On your Mac, connect a model, or check which models it may use with omnillm hub devices (omnillm hub device set NAME --all-models allows all)."))
}
