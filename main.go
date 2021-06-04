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
	"sync"
	"time"
)

var (
	myWindow fyne.Window
	//
	lock *safeLock
	//lock   = false
	failed bool
	//
	conn = false
	//
	msgRec     *widget.Entry
	inputAddr  *widget.Entry
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

type safeLock struct {
	mu   sync.Mutex
	lock bool
}

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
	lock = &safeLock{}
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
		if !conn {
			dialog.NewError(errors.New("you have to connect to the server first"), myWindow).Show()
			return
		}
		_ = client.SetWriteDeadline(time.Now().Add(5 * time.Second))
		err := client.WriteMessage(websocket.TextMessage, []byte(msgToSend.Text))
		if err != nil {
			dialog.NewError(errors.New("send msg failed"), myWindow).Show()
			//panic("send msg failed")
		}
		msgToSend.SetText("")
		msgRec.SetText("")
	}
	//shortcutBtn := widget.NewButton("Shortcut", nil)
	h2 := container.New(layout.NewVBoxLayout(), connectBtn)
	card := widget.NewCard("Websocket Client", "A tool made by Starx", container.New(layout.NewVBoxLayout(), inputAddr, msgRec, msgToSend, h2))
	myWindow.SetContent(card)
	myWindow.ShowAndRun()
}

func connect() {
	if lock.lock {
		fmt.Println("wtf")
		return
	}
	if !conn {
		if len(inputAddr.Text) == 0 {
			dialog.NewError(errors.New("why not to write something"), myWindow).Show()
			return
		}
		inputAddr.SetText(strings.ReplaceAll(inputAddr.Text, " ", ""))
		failed = false
		done = make(chan struct{})
		_, err := url.Parse(inputAddr.Text)
		if err != nil {
			dialog.NewError(errors.New("parse server address failed"), myWindow).Show()
			return
			//panic("parse server address failed")
		}
		lock.mu.Lock()
		lock.lock = true
		client, _, err = para.Dial(inputAddr.Text, nil)
		lock.lock = false
		lock.mu.Unlock()
		if err != nil {
			dialog.NewError(errors.New("dial to server failed"), myWindow).Show()
			return
			//panic("dial to server failed")
		}
		fmt.Println("ws connect ok")
		conn = true
		connectBtn.SetText("DisConnect")
		defer client.Close()
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
				fmt.Printf("recv: %s", message)
				fmt.Println()
				msgRec.SetText(msgRec.Text + string(message) + "\n")
			}
		}()

		for conn {
			select {
			case <-done:
				fmt.Println("done")
				conn = false
				connectBtn.SetText("Connect")
				return
			case <-interrupt:
				fmt.Println("interrupt")
				conn = false
				err := client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					fmt.Println("write close:", err)
				}
				return
			}
		}
	} else {
		conn = false
		if !failed {
			done <- struct{}{}
		}
		msgRec.SetText("")
	}
}
