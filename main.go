package main

import (
	"flag"
	"strings"

	"encoding/json"

	"github.com/asticode/go-astilectron"
	"github.com/asticode/go-astilectron-bootstrap"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

// AppInfo given by the front end.
type AppInfo struct {
	Peer1    string
	Peer2    string
	FileName string
}

// Vars
var (
	AppName string
	BuiltAt string
	debug   = flag.Bool("d", true, "enables the debug mode")
	w       *astilectron.Window
)

type MessageOut struct {
	Name    string      `json:"name"`
	Payload interface{} `json:"payload,omitempty"`
}

func main() {
	// Init
	flag.Parse()
	astilog.FlagInit()

	// Run bootstrap
	astilog.Debugf("Running app built at %s", BuiltAt)
	options := bootstrap.Options{
		Asset: Asset,
		AstilectronOptions: astilectron.Options{
			AppName:            AppName,
			AppIconDarwinPath:  "resources/images/icon.icns",
			AppIconDefaultPath: "resources/images/icon.png",
		},
		MenuOptions:    []*astilectron.MenuItemOptions{{}},
		Debug:          *debug,
		Homepage:       "index.html",
		MessageHandler: handleMessages,
		RestoreAssets:  RestoreAssets,
		WindowOptions: &astilectron.WindowOptions{
			BackgroundColor: astilectron.PtrStr("#333"),
			Center:          astilectron.PtrBool(true),
			Height:          astilectron.PtrInt(700),
			Width:           astilectron.PtrInt(700),
		},
	}
	err := bootstrap.Run(options)

	if err != nil {
		astilog.Fatal(errors.Wrap(err, "running bootstrap failed"))
	}
}

// handleMessages handles messages
func handleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	info := new(AppInfo)
	var msg string
	if len(m.Payload) > 0 {
		err = json.Unmarshal(m.Payload, &msg)
		if err != nil {
			payload = err.Error()
			return
		}
		err = json.Unmarshal([]byte(msg), &info)
		if err != nil {
			payload = err.Error()
			return
		}
		go initTransfer(info.Peer1, info.Peer2, info.FileName)
		payload = info.FileName
	}
	return
}

func notifyFrontEnd(msg string) {
	if strings.Contains(msg, "finished") {
		bootstrap.SendMessage(w, "Finished", msg, func(m *bootstrap.MessageIn) {
		})
	} else if strings.Contains(msg, "Connected") {
		bootstrap.SendMessage(w, "Connected", msg, func(m *bootstrap.MessageIn) {
		})
	} else {
		bootstrap.SendMessage(w, "error", msg, func(m *bootstrap.MessageIn) {
		})
	}
}
