package butler

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type CloseMessage struct {
	CloseErr error
}

type sendMessage struct {
	data  []byte
	mtype int
}

type Websocket struct {
	conn         *websocket.Conn
	open         bool
	openMx       sync.RWMutex
	sendChannel  chan sendMessage
	done         chan bool
	readingStart sync.Once
	shutdownOnce sync.Once
	logger       RequestLogger
	request      *Request

	msgReceivers   eventEmitter[WebsocketMessage]
	closeReceivers eventEmitter[CloseMessage]
}

func newWebsocket(request *Request, conn *websocket.Conn, pingInterval, pongTimeout, writeTimeout time.Duration) *Websocket {
	conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongTimeout)); return nil })

	ws := Websocket{
		conn:        conn,
		open:        true,
		openMx:      sync.RWMutex{},
		sendChannel: make(chan sendMessage, 16),
		done:        make(chan bool),
		logger:      request.Logger,
		request:     request,
		msgReceivers: eventEmitter[WebsocketMessage]{
			mx:        sync.Mutex{},
			listeners: make([]listener[WebsocketMessage], 0),
		},
		closeReceivers: eventEmitter[CloseMessage]{
			mx:        sync.Mutex{},
			listeners: make([]listener[CloseMessage], 0),
		},
	}

	ws.closeReceivers.Add(func(CloseMessage) {
		ws.openMx.Lock()
		ws.open = false
		ws.openMx.Unlock()
	})

	orgCloseHandler := ws.conn.CloseHandler()
	ws.conn.SetCloseHandler(func(code int, text string) error {
		ws.closeReceivers.EmitAndClose(CloseMessage{
			CloseErr: fmt.Errorf("code=%v %s", code, text),
		})
		return orgCloseHandler(code, text)
	})

	// writer
	go func() {
		var err error

		ticker := time.NewTicker(pingInterval)
		defer func() {
			ticker.Stop()
			ws.shutdown(err)
		}()

		for {
			select {
			case msg := <-ws.sendChannel:
				conn.SetWriteDeadline(time.Now().Add(writeTimeout))

				var w io.WriteCloser
				w, err = conn.NextWriter(msg.mtype)
				if err != nil {
					request.Logger.Error(err)
					return
				}
				w.Write(msg.data)

				err = w.Close()
				if err != nil {
					request.Logger.Error(err)
					return
				}
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(writeTimeout))
				if err = conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					request.Logger.Error(err)
					return
				}
			case <-ws.done:
				return
			}
		}
	}()

	return &ws
}

func (ws *Websocket) startReading() {
	ws.readingStart.Do(func() {
		go func() {
			for {
				t, content, err := ws.conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						ws.logger.Error(err)
					}
					ws.shutdown(err)
					return
				}
				ws.msgReceivers.Emit(WebsocketMessage{
					IsText: t == websocket.TextMessage,
					Data:   content,
				})
			}
		}()
	})
}

func (ws *Websocket) start(onOpen func(ws *Websocket) error) {
	err := onOpen(ws)
	if err != nil {
		ws.logger.Error("the WebSocket open handler returned an error:", err)
		if ws.IsOpen() {
			ws.Close(websocket.CloseInternalServerErr)
		}
	} else {
		if ws.IsOpen() {
			var wg sync.WaitGroup
			wg.Add(1)
			ws.closeReceivers.Add(func(CloseMessage) {
				wg.Done()
			})
			ws.startReading()
			wg.Wait()
		}
	}
}

func (ws *Websocket) GetConnection() *websocket.Conn {
	return ws.conn
}

func (ws *Websocket) Request() *Request {
	return ws.request
}

func (ws *Websocket) IsOpen() bool {
	ws.openMx.RLock()
	defer ws.openMx.RUnlock()
	return ws.open
}

// optionally pass a one close code. default is `websocket.CloseNormalClosure`
func (ws *Websocket) shutdown(err error) {
	ws.shutdownOnce.Do(func() {
		close(ws.done)
	})
	ws.conn.Close()
	ws.closeReceivers.EmitAndClose(CloseMessage{
		CloseErr: err,
	})
}

func (ws *Websocket) Close(closecode ...int) error {
	code := websocket.CloseNormalClosure

	if len(closecode) > 0 {
		code = closecode[0]
	}

	ws.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, ""),
		time.Now().Add(time.Second),
	)

	err := ws.conn.Close()
	ws.shutdown(err)
	return err
}

// Add a listener for the ws messages, given function will be called whenever a new message is received.
//
// Returns a function that can be called to remove the listener.
func (ws *Websocket) OnMessage(handler func(msg WebsocketMessage)) func() {
	ws.startReading()

	id := ws.msgReceivers.Add(handler)
	return func() {
		ws.msgReceivers.Remove(id)
	}
}

// Add a listener for the ws close, given function will be called once the websocket is closed.
//
// Returns a function that can be called to remove the listener.
func (ws *Websocket) OnClose(handler func(closeMsg CloseMessage)) func() {
	id := ws.closeReceivers.Add(handler)
	return func() {
		ws.closeReceivers.Remove(id)
	}
}

// returns the next read msg, if the connection closes before any msg is read - it will return nil
func (ws *Websocket) ReadNext() *WebsocketMessage {
	var removeMsgListener func()
	var removeCloseListener func()

	msgChan := make(chan WebsocketMessage, 1)
	closeChan := make(chan CloseMessage, 1)

	defer func() {
		removeCloseListener()
		removeMsgListener()
		close(msgChan)
		close(closeChan)
	}()

	removeMsgListener = ws.OnMessage(func(msg WebsocketMessage) {
		msgChan <- msg
	})

	removeCloseListener = ws.OnClose(func(msg CloseMessage) {
		closeChan <- msg
	})

	select {
	case msg := <-msgChan:
		return &msg
	case <-closeChan:
		return nil
	}
}

func (ws *Websocket) SendBinary(content []byte) bool {
	select {
	case ws.sendChannel <- sendMessage{content, websocket.BinaryMessage}:
		return true
	case <-ws.done:
		return false
	}
}

func (ws *Websocket) SendText(content string) bool {
	select {
	case ws.sendChannel <- sendMessage{[]byte(content), websocket.TextMessage}:
		return true
	case <-ws.done:
		return false
	}

}

func (ws *Websocket) Logger() RequestLogger {
	return ws.logger
}

type WebsocketMessage struct {
	IsText bool
	Data   []byte
}

func (msg WebsocketMessage) Text() string {
	return string(msg.Data)
}

func (msg WebsocketMessage) JsonBind(to any) error {
	return json.Unmarshal(msg.Data, to)
}
