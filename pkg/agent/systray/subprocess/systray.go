package subprocess

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/getlantern/systray"
	api "github.com/manifold/tractor/pkg/agent/systray"
	iconData "github.com/manifold/tractor/pkg/data/icons"
)

var (
	inbox = make(chan api.Message)
	icons = map[string][]byte{
		"tractor":     iconData.Tractor,
		"unavailable": iconData.Unavailable,
		"available":   iconData.Available,
		"partially":   iconData.Partially,
	}
)

func Run() {
	go receiveMessages(inbox, os.Stdin)
	systray.Run(onReady, nil)
}

func sendMessage(msg api.Message) {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(append(b, '\n'))
}

func sendError(err error) {
	text := err.Error()
	sendMessage(api.Message{
		Type:  api.Error,
		Error: &text,
	})
}

func receiveMessages(ch chan api.Message, r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var msg api.Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			sendError(err)
		}
		ch <- msg
	}
	if err := scanner.Err(); err != nil {
		sendError(err)
	}
}

func onReady() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigCh
		systray.Quit()
	}()
	for msg := range inbox {
		switch msg.Type {
		case api.InitMenu:
			initMenu(msg.Menu)
		default:
			// TODO?
		}
	}
}

func initMenu(menu *api.Menu) {
	if menu.Icon != "" {
		systray.SetIcon(icons[menu.Icon])
	}
	systray.SetTitle(menu.Title)
	systray.SetTooltip(menu.Tooltip)

	for _, item := range menu.Items {
		if item.Title == "-" {
			systray.AddSeparator()
			continue
		}
		menuItem := systray.AddMenuItem(item.Title, item.Tooltip)
		if item.Checked {
			menuItem.Check()
		} else {
			menuItem.Uncheck()
		}
		if item.Enabled {
			menuItem.Enable()
		} else {
			menuItem.Disable()
		}
		if item.Icon != "" {
			menuItem.SetIcon(icons[item.Icon])
		}
		go func(menuItem *systray.MenuItem, item api.MenuItem) {
			for range menuItem.ClickedCh {
				sendMessage(api.Message{
					Type: api.ItemClicked,
					Item: &item,
				})
			}
		}(menuItem, item)
	}
}
