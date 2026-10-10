package butler_test

import (
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	f "github.com/ncpa0cpl/butler"
	"github.com/stretchr/testify/assert"
)

func startWsServer(t *testing.T, endpoint *f.WebSocketEndpoint) {
	server := f.CreateServer()
	server.Port = 8081
	server.Add(endpoint)

	go server.Listen()
	t.Cleanup(server.Close)
}

func readUntilError(conn *websocket.Conn) *websocket.CloseError {
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok {
				return closeErr
			}
			return nil
		}
	}
}

func TestWebSocketEndpointEcho(t *testing.T) {
	assert := assert.New(t)

	closeObserved := make(chan struct{}, 1)

	echo := &f.WebSocketEndpoint{
		Path: "/ws/echo",
		OnOpen: func(ws *f.Websocket) error {
			assert.True(ws.IsOpen())
			assert.NotNil(ws.Request())
			assert.NotNil(ws.Logger())

			assert.True(ws.SendText("welcome"))
			assert.True(ws.SendBinary([]byte{0x01, 0x02}))

			ws.OnMessage(func(msg f.WebsocketMessage) {
				if msg.IsText {
					ws.SendText("echo: " + msg.Text())
				} else {
					ws.SendBinary(msg.Data)
				}
			})

			ws.OnClose(func(_ f.CloseMessage) {
				closeObserved <- struct{}{}
			})

			return nil
		},
	}

	startWsServer(t, echo)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8081/ws/echo", nil)
	noErr(err)
	defer conn.Close()

	mtype, data, err := conn.ReadMessage()
	noErr(err)
	assert.Equal(websocket.TextMessage, mtype)
	assert.Equal("welcome", string(data))

	mtype, data, err = conn.ReadMessage()
	noErr(err)
	assert.Equal(websocket.BinaryMessage, mtype)
	assert.Equal([]byte{0x01, 0x02}, data)

	noErr(conn.WriteMessage(websocket.TextMessage, []byte("hello")))
	mtype, data, err = conn.ReadMessage()
	noErr(err)
	assert.Equal(websocket.TextMessage, mtype)
	assert.Equal("echo: hello", string(data))

	noErr(conn.WriteMessage(websocket.BinaryMessage, []byte{0xDE, 0xAD}))
	mtype, data, err = conn.ReadMessage()
	noErr(err)
	assert.Equal(websocket.BinaryMessage, mtype)
	assert.Equal([]byte{0xDE, 0xAD}, data)

	noErr(conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	))

	select {
	case <-closeObserved:
	case <-time.After(5 * time.Second):
		t.Fatal("OnClose handler was not called")
	}
}

func TestWebSocketEndpointReadNext(t *testing.T) {
	assert := assert.New(t)

	closeObserved := make(chan struct{}, 1)

	endpoint := &f.WebSocketEndpoint{
		Path: "/ws/readnext",
		OnOpen: func(ws *f.Websocket) error {
			go func() {
				for {
					msg := ws.ReadNext()
					if msg == nil {
						closeObserved <- struct{}{}
						return
					}
					ws.SendText("echo: " + msg.Text())
				}
			}()
			return nil
		},
	}

	startWsServer(t, endpoint)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8081/ws/readnext", nil)
	noErr(err)

	noErr(conn.WriteMessage(websocket.TextMessage, []byte("hi")))
	_, data, err := conn.ReadMessage()
	noErr(err)
	assert.Equal("echo: hi", string(data))

	noErr(conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	))

	select {
	case <-closeObserved:
	case <-time.After(5 * time.Second):
		t.Fatal("ReadNext did not return nil after the connection was closed")
	}
}

func TestWebSocketEndpointPongTimeout(t *testing.T) {
	assert := assert.New(t)

	closeObserved := make(chan struct{}, 1)
	wsCh := make(chan *f.Websocket, 1)
	var openAtClose atomic.Bool

	endpoint := &f.WebSocketEndpoint{
		Path:        "/ws/pongtimeout",
		PongTimeout: 300 * time.Millisecond,
		OnOpen: func(ws *f.Websocket) error {
			wsCh <- ws
			ws.OnClose(func(_ f.CloseMessage) {
				openAtClose.Store(ws.IsOpen())
				closeObserved <- struct{}{}
			})
			return nil
		},
	}

	startWsServer(t, endpoint)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8081/ws/pongtimeout", nil)
	noErr(err)
	defer conn.Close()

	// ignore pings so the server sees the connection as dead
	conn.SetPingHandler(func(string) error { return nil })

	ws := <-wsCh
	assert.True(ws.IsOpen())

	readUntilError(conn)

	select {
	case <-closeObserved:
	case <-time.After(5 * time.Second):
		t.Fatal("connection was not closed after the pong timeout")
	}

	assert.False(openAtClose.Load())
	assert.False(ws.IsOpen())

	// the send buffer may still accept a few sends after the close, so keep
	// trying
	sendFailed := false
	for i := 0; i < 100; i++ {
		if !ws.SendText("too late") {
			sendFailed = true
			break
		}
	}
	assert.True(sendFailed)
}

func TestWebSocketEndpointPingsKeepAlive(t *testing.T) {
	closeObserved := make(chan struct{}, 1)

	endpoint := &f.WebSocketEndpoint{
		Path: "/ws/alive",
		// pings reset the read deadline only if the client answers them;
		// without that the connection would die after the pong timeout
		PingInterval: 20 * time.Millisecond,
		PongTimeout:  300 * time.Millisecond,
		OnOpen: func(ws *f.Websocket) error {
			ws.OnClose(func(_ f.CloseMessage) {
				closeObserved <- struct{}{}
			})
			return nil
		},
	}

	startWsServer(t, endpoint)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8081/ws/alive", nil)
	noErr(err)
	defer conn.Close()

	// the gorilla client answers pings automatically; a read timeout here
	// means the connection survived the pong timeout
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, _, err = conn.ReadMessage()
	if err != nil {
		ne, isTimeout := err.(net.Error)
		if !isTimeout || !ne.Timeout() {
			t.Fatalf("connection closed while pings were being answered: %v", err)
		}
	}

	noErr(conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	))

	select {
	case <-closeObserved:
	case <-time.After(5 * time.Second):
		t.Fatal("OnClose handler was not called")
	}
}

func TestWebSocketEndpointOnOpenError(t *testing.T) {
	assert := assert.New(t)

	endpoint := &f.WebSocketEndpoint{
		Path: "/ws/error",
		OnOpen: func(ws *f.Websocket) error {
			return errors.New("boom")
		},
	}

	startWsServer(t, endpoint)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8081/ws/error", nil)
	noErr(err)
	defer conn.Close()

	closeErr := readUntilError(conn)
	if assert.NotNil(closeErr) {
		assert.Equal(websocket.CloseInternalServerErr, closeErr.Code)
	}
}

func TestWebSocketEndpointBeforeUpgrade(t *testing.T) {
	assert := assert.New(t)

	endpoint := &f.WebSocketEndpoint{
		Path: "/ws/rejected",
		BeforeUpgrade: func(request *f.Request) *f.Response {
			return f.Respond.Unauthorized().Text("no upgrade for you")
		},
		OnOpen: func(ws *f.Websocket) error {
			return nil
		},
	}

	startWsServer(t, endpoint)

	body, resp := request("GET", "http://localhost:8081/ws/rejected", nil)
	assert.Equal(401, resp.StatusCode)
	assert.Equal("no upgrade for you", string(body))
}

func TestWebSocketEndpointSendAfterClose(t *testing.T) {
	assert := assert.New(t)

	var sendLoopDone atomic.Bool
	var sawFailedSend atomic.Bool

	endpoint := &f.WebSocketEndpoint{
		Path: "/ws/spam",
		OnOpen: func(ws *f.Websocket) error {
			go func() {
				defer sendLoopDone.Store(true)

				for i := 0; i < 1_000_000; i++ {
					if !ws.SendText("spam") || !ws.SendBinary([]byte{0}) {
						sawFailedSend.Store(true)
						return
					}
				}
			}()

			// fill the channel buffer so sends are blocked when the connection closes
			time.Sleep(50 * time.Millisecond)

			ws.Close(websocket.CloseGoingAway)
			return nil
		},
	}

	startWsServer(t, endpoint)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8081/ws/spam", nil)
	noErr(err)
	defer conn.Close()

	closeErr := readUntilError(conn)
	if assert.NotNil(closeErr) {
		assert.Equal(websocket.CloseGoingAway, closeErr.Code)
	}

	waitUntil(func() bool {
		return sendLoopDone.Load()
	})

	assert.True(
		sawFailedSend.Load(),
		"SendText/SendBinary should report failure once the websocket is closed",
	)
}
