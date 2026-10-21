module github.com/wensuper/OmniLLM

go 1.27.2

require (
	github.com/coder/websocket v1.8.15
	github.com/flynn/noise v1.1.0
	github.com/makiuchi-d/gozxing v0.1.1
	github.com/zalando/go-keyring v0.2.8
	golang.org/x/crypto v0.58.0
	golang.org/x/image v0.47.0
	golang.org/x/term v0.47.0
	rsc.io/qr v0.2.0
)

require (
	fyne.io/systray v1.12.2 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)

replace fyne.io/systray => ./third_party/systray
