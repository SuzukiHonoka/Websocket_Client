package main

import (
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"
)

var (
	myWindow fyne.Window
	//
	failed bool
	//
	conn bool
	//
	msgRec     *widget.Entry
	inputAddr  *KeyDownEntryEx
	connectBtn *widget.Button
	//
	para = &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 3 * time.Second,
	}
	client *websocket.Conn
	//
	interrupt = make(chan os.Signal, 1)
	done      = make(chan struct{})
	//
)

type KeyDownEntryEx struct {
	widget.Entry
	action func()
}

func (e *KeyDownEntryEx) KeyUp(key *fyne.KeyEvent) {
	if e.Disabled() {
		return
	}
	if e.action != nil {
		if key.Name == fyne.KeyEnter || key.Name == fyne.KeyReturn {
			e.action()
			fmt.Println("key down")
		}
	}
}

func NewKeyDownEntryEx(multi bool) *KeyDownEntryEx {
	entry := &KeyDownEntryEx{Entry: widget.Entry{MultiLine: multi, Wrapping: fyne.TextTruncate}}
	entry.ExtendBaseWidget(entry)
	return entry
}

func main() {
	signal.Notify(interrupt, os.Interrupt)
	//
	myApp := app.New()
	myApp.Settings().SetTheme(theme.LightTheme())
	myWindow = myApp.NewWindow("Websocket_Client_by_Starx")
	myWindow.Resize(fyne.Size{
		Width:  600,
		Height: 300,
	})
	msgRec = widget.NewMultiLineEntry()
	msgRec.SetPlaceHolder("Msg received")
	inputAddr = NewKeyDownEntryEx(false)
	inputAddr.SetPlaceHolder("Enter server address...")
	inputAddr.action = proxy
	connectBtn = widget.NewButton("Connect", proxy)
	msgToSend := widget.NewMultiLineEntry()
	msgToSend.SetPlaceHolder("Press enter to send message")
	sendMsg := func() {
		if !conn {
			dialog.NewError(errors.New("you have to connect to the server first"), myWindow).Show()
			return
		}
		_ = client.SetWriteDeadline(time.Now().Add(5 * time.Second))
		err := client.WriteMessage(websocket.TextMessage, []byte(msgToSend.Text))
		if err != nil {
			dialog.NewError(errors.New("send msg failed"), myWindow).Show()
		}
		msgToSend.SetText("")
		//msgRec.SetText("")
	}
	send := widget.NewButton("Send", sendMsg)
	//shortcutBtn := widget.NewButton("Shortcut", nil)
	//h2 := container.New(layout.NewVBoxLayout(), connectBtn)
	card := widget.NewCard("Websocket Client", "A tool made by Starx", container.New(layout.NewVBoxLayout(), inputAddr, connectBtn, msgRec, msgToSend, send))
	myWindow.SetContent(card)
	myWindow.ShowAndRun()
	go func() {
		for {
			select {
			case <-interrupt:
				if conn {
					disconnect()
				}
				fmt.Println("interrupt")
				os.Exit(1)
			}
		}
	}()
}

func proxy() {
	go connect()
}

func connect() {
	if connectBtn.Disabled() {
		return
	}
	if !conn {
		connectBtn.Disable()
		msgRec.SetText("")
		if len(inputAddr.Text) == 0 {
			dialog.NewError(errors.New("why not to write something"), myWindow).Show()
			return
		}
		inputAddr.SetText(strings.TrimSpace(inputAddr.Text))
		failed = false
		done = make(chan struct{})
		_, err := url.Parse(inputAddr.Text)
		if err != nil {
			dialog.NewError(errors.New("parse server address failed"), myWindow).Show()
			return
		}
		connectBtn.SetText("Connecting")
		client, _, err = para.Dial(inputAddr.Text, nil)
		if err != nil {
			dialog.NewError(errors.New("dial to server failed:\n"+err.Error()), myWindow).Show()
			connectBtn.Enable()
			connectBtn.SetText("Connect")
			return
		}
		fmt.Println("ws connect ok")
		conn = true
		connectBtn.SetText("DisConnect")
		connectBtn.Enable()
		defer func(client *websocket.Conn) {
			err := client.Close()
			if err != nil {
				fmt.Println(err.Error())
			}
		}(client)
		go func() {
			defer close(done)
			for conn {
				fmt.Println("reading..")
				_, message, err := client.ReadMessage()
				if err != nil {
					failed = true
					conn = false
					if !conn {
						return
					}
					dialog.NewError(err, myWindow).Show()
					return
				}
				fmt.Printf("recv: %s\n", message)
				msgRec.SetText(msgRec.Text + string(message) + "\n")
			}
		}()
		for conn {
			select {
			case <-done:
				disconnect()
				connectBtn.SetText("Connect")
				connectBtn.Enable()
				fmt.Println("done")
			}
		}
		return
	}
	if failed {
		conn = false
		return
	}
	done <- struct{}{}
}

func disconnect() {
	err := client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		fmt.Println("write close:", err)
	}
}
