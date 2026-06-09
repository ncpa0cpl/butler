package butler

import (
	"encoding/json"
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
	conn           *websocket.Conn
	open           bool
	sendChannel    chan sendMessage
	readingStarted bool
	logger         RequestLogger
	request        *Request

	msgReceivers   eventEmitter[WebsocketMessage]
	closeReceivers eventEmitter[CloseMessage]
}

func newWebsocket(request *Request, conn *websocket.Conn, pingInterval, pongTimeout, writeTimeout time.Duration) *Websocket {
	conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongTimeout)); return nil })

	ws := Websocket{
		conn:           conn,
		open:           true,
		sendChannel:    make(chan sendMessage, 16),
		logger:         request.Logger,
		readingStarted: false,
		request:        request,
		msgReceivers: eventEmitter[WebsocketMessage]{
			mx:        sync.Mutex{},
			listeners: make([]listener[WebsocketMessage], 0),
		},
		closeReceivers: eventEmitter[CloseMessage]{
			mx:        sync.Mutex{},
			listeners: make([]listener[CloseMessage], 0),
		},
	}

	var mutex sync.Mutex
	ws.closeReceivers.Add(func(CloseMessage) {
		mutex.Lock()
		ws.open = false
		mutex.Unlock()
	})

	// writer
	go func() {
		ticker := time.NewTicker(pingInterval)
		defer func() {
			close(ws.sendChannel)
			ticker.Stop()
			ws.conn.Close()
		}()

		for {
			select {
			case msg, ok := <-ws.sendChannel:
				conn.SetWriteDeadline(time.Now().Add(writeTimeout))
				if !ok {
					// The hub closed the channel.
					err := conn.WriteMessage(websocket.CloseMessage, []byte{})
					if err != nil {
						request.Logger.Error(err)
					}
					return
				}

				w, err := conn.NextWriter(msg.mtype)
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
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					request.Logger.Error(err)
					return
				}
			}
		}
	}()

	return &ws
}

func (ws *Websocket) startReading() {
	if ws.readingStarted {
		return
	}

	ws.readingStarted = true

	go func() {
		for ws.open {
			t, content, err := ws.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					ws.logger.Error(err)
				}
				ws.closeReceivers.EmitAndClose(CloseMessage{
					err,
				})
				break
			}
			ws.msgReceivers.Emit(WebsocketMessage{
				IsText: t == websocket.TextMessage,
				Data:   content,
			})
		}
	}()
}

func (ws *Websocket) GetConnection() *websocket.Conn {
	return ws.conn
}

func (ws *Websocket) Request() *Request {
	return ws.request
}

func (ws *Websocket) IsOpen() bool {
	return ws.open
}

func (ws *Websocket) Close() error {
	err := ws.conn.Close()
	ws.closeReceivers.EmitAndClose(CloseMessage{
		err,
	})
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
		close(msgChan)
		close(closeChan)
		removeCloseListener()
		removeMsgListener()
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

func (ws *Websocket) SendBinary(content []byte) {
	ws.sendChannel <- sendMessage{content, websocket.BinaryMessage}
}

func (ws *Websocket) SendText(content string) {
	ws.sendChannel <- sendMessage{[]byte(content), websocket.TextMessage}
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
