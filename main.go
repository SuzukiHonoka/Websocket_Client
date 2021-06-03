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
	"net/url"
	"os"
	"os/signal"
)

var (
	myWindow fyne.Window
	//
	conn = false
	//
	msgRec     *widget.Entry
	inputAddr  *widget.Entry
	connectBtn *widget.Button
	//
	client    *websocket.Conn
	interrupt = make(chan os.Signal, 1)
	done      = make(chan struct{})
	//
)

type multiLineEntryEx struct {
	widget.Entry
	action func()
}

func (e *multiLineEntryEx) KeyUp(key *fyne.KeyEvent) {
	if e.Disabled() {
		return
	}
	if e.action != nil {
		if key.Name == fyne.KeyEnter || key.Name == fyne.KeyReturn {
			e.action()
		}
	}
}

func NewMultiLineEntryEx() *multiLineEntryEx {
	entry := &multiLineEntryEx{Entry: widget.Entry{MultiLine: true, Wrapping: fyne.TextTruncate}}
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
	inputAddr = widget.NewEntry()
	connectBtn = widget.NewButton("Connect", func() {
		go connect()
	})
	inputAddr.SetPlaceHolder("Enter server address...")
	msgRec.SetPlaceHolder("Msg received")
	msgToSend := NewMultiLineEntryEx()
	msgToSend.SetPlaceHolder("Press enter to send message")
	msgToSend.action = func() {
		msgToSend.SetText("")
		if !conn {
			dialog.NewError(errors.New("you have to connect to the server first"), myWindow).Show()
			return
		}
		err := client.WriteMessage(websocket.TextMessage, []byte(msgToSend.Text))
		if err != nil {
			dialog.NewError(errors.New("send msg failed"), myWindow).Show()
			//panic("send msg failed")
		}
		msgRec.SetText("")
	}
	//shortcutBtn := widget.NewButton("Shortcut", nil)
	h2 := container.New(layout.NewVBoxLayout(), connectBtn)
	card := widget.NewCard("Websocket Client", "A tool made by Starx", container.New(layout.NewVBoxLayout(), inputAddr, msgRec, msgToSend, h2))
	myWindow.SetContent(card)
	myWindow.ShowAndRun()
}

func connect() {
	if !conn {
		if len(inputAddr.Text) == 0 {
			dialog.NewError(errors.New("why not to write something"), myWindow).Show()
			return
		}
		_, err := url.Parse(inputAddr.Text)
		if err != nil {
			dialog.NewError(errors.New("parse server address failed"), myWindow).Show()
			//panic("parse server address failed")
		}
		client, _, err = websocket.DefaultDialer.Dial(inputAddr.Text, nil)
		if err != nil {
			dialog.NewError(errors.New("dial to server failed"), myWindow).Show()
			return
			//panic("dial to server failed")
		}
		fmt.Println("ws connect ok")
		conn = !conn
		connectBtn.SetText("DisConnect")
		defer client.Close()
		go func() {
			defer close(done)
			for {
				_, message, err := client.ReadMessage()
				if err != nil {
					dialog.NewError(err, myWindow).Show()
					return
				}
				fmt.Printf("recv: %s", message)
				msgRec.SetText(msgRec.Text + string(message) + "\n")
			}
		}()
		for {
			select {
			case <-done:
				connectBtn.SetText("Connect")
			case <-interrupt:
				fmt.Println("interrupt")
				err := client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					fmt.Println("write close:", err)
					return
				}
			}
		}
	} else {
		conn = !conn
		done <- struct{}{}
		msgRec.SetText("")
	}
}
