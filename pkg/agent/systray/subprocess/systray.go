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
		"Unavailable": iconData.Unavailable,
		"Available":   iconData.Available,
		"Partially":   iconData.Partially,
	}

	menuItems []*systray.MenuItem
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
		case api.ItemUpdate:
			updateItem(msg.Idx, msg.Item)
		default:
			// TODO?
		}
	}
}

func updateItem(idx int, item *api.MenuItem) {
	if idx < 0 || idx > len(menuItems)-1 {
		return
	}
	applyItem(menuItems[idx], item)
}

func applyItem(i *systray.MenuItem, ii *api.MenuItem) {
	i.SetTitle(ii.Title)
	i.SetTooltip(ii.Tooltip)
	if ii.Checked {
		i.Check()
	} else {
		i.Uncheck()
	}
	if ii.Enabled {
		i.Enable()
	} else {
		i.Disable()
	}
	if ii.Icon != "" {
		i.SetIcon(icons[ii.Icon])
	}
}

func initMenu(menu *api.Menu) {
	menuItems = nil

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
		menuItems = append(menuItems, menuItem)
		applyItem(menuItem, &item)
		go func(menuItem *systray.MenuItem, item api.MenuItem) {
			for range menuItem.ClickedCh {
				// FIXME: this will only send the original item, even if the item updates
				sendMessage(api.Message{
					Type: api.ItemClicked,
					Item: &item,
				})
			}
		}(menuItem, item)
	}
}
