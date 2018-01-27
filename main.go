package main

import (
	"encoding/json"
	"flag"
	"strings"

	"github.com/asticode/go-astilectron"
	bootstrap "github.com/asticode/go-astilectron-bootstrap"
	astilog "github.com/asticode/go-astilog"
	"github.com/pkg/errors"
)

const about string = "This is a simple P2P File transferer application that allows users to transfer files at extremely fast speeds using Googles quic protocol!"

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
	debug   = flag.Bool("d", false, "enables the debug mode")
	w       *astilectron.Window
)

// MessageOut used to recieve information from frontEnd
type MessageOut struct {
	Name    string      `json:"name"`
	Payload interface{} `json:"payload"`
}

func main() {
	// Init
	flag.Parse()
	astilog.FlagInit()

	// Run bootstrap
	astilog.Debugf("Running app built at %s", BuiltAt)
	if err := bootstrap.Run(bootstrap.Options{
		Asset: Asset,
		AstilectronOptions: astilectron.Options{
			AppName:            AppName,
			AppIconDarwinPath:  "resources/images/icon.icns",
			AppIconDefaultPath: "resources/images/icon.png",
		},
		Debug:    *debug,
		Homepage: "index.html",
		MenuOptions: []*astilectron.MenuItemOptions{{
			Label: astilectron.PtrStr("File"),
			SubMenu: []*astilectron.MenuItemOptions{
				{
					Label: astilectron.PtrStr("About"),
					OnClick: func(e astilectron.Event) (deleteListener bool) {
						if err := bootstrap.SendMessage(w, "about", about, func(m *bootstrap.MessageIn) {
							var s string
							if err := json.Unmarshal(m.Payload, &s); err != nil {
								astilog.Error(errors.Wrap(err, "unmarshaling payload failed"))
								return
							}
							astilog.Infof("About modal has been displayed and payload is %s!", s)
						}); err != nil {
							astilog.Error(errors.Wrap(err, "sending about event failed"))
						}
						return
					},
				},
				{Role: astilectron.MenuItemRoleClose},
			},
		}},
		OnWait: func(_ *astilectron.Astilectron, iw *astilectron.Window, _ *astilectron.Menu, _ *astilectron.Tray, _ *astilectron.Menu) error {
			w = iw
			go func() {
				if err := bootstrap.SendMessage(w, "check.out.menu", "WE ARE SENDING"); err != nil {
					astilog.Error(errors.Wrap(err, "sending check.out.menu event failed"))
				}
			}()
			return nil
		},
		MessageHandler: handleMessages,
		RestoreAssets:  RestoreAssets,
		WindowOptions: &astilectron.WindowOptions{
			BackgroundColor: astilectron.PtrStr("#333"),
			Center:          astilectron.PtrBool(true),
			Height:          astilectron.PtrInt(700),
			Width:           astilectron.PtrInt(700),
		},
	}); err != nil {
		astilog.Fatal(errors.Wrap(err, "running bootstrap failed"))
	}
}

// handleMessages handles messages
func handleMessages(_ *astilectron.Window, m bootstrap.MessageIn) (payload interface{}, err error) {
	info := new(AppInfo)
	var msg string
	info.FileName = ""
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
		payload = "recieved"
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
		bootstrap.SendMessage(w, "Error", msg, func(m *bootstrap.MessageIn) {
		})
	}
}
