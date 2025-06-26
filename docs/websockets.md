# WebSockets

Butler provides a very simple and easy to use api for creating WebSocket connections.

Handling WS messages can be done in two different ways:

### Through continuous reading via ReadNext() method

```go
package main

import "github.com/ncpa0cpl/butler"

var WsLoopbackEndpoint = butler.WebSocketEndpoint{
	Path: "/ws_loopback",
	OnOpen: func(ws *butler.Websocket) error {
		for nextMsg := ws.ReadNext(); nextMsg != nil; nextMsg = ws.ReadNext() {
			// handle received message
			ws.SendText(nextMsg.Text()) // send back what was received
		}

		// nextMsg == nil means the connection has closed

		return nil
	},
}
```

### Through a event listener

```go
package main

import "github.com/ncpa0cpl/butler"

var WsLoopbackEndpoint = butler.WebSocketEndpoint{
	Path: "/ws_loopback",
	OnOpen: func(ws *butler.Websocket) error {

		ws.OnMessage(func(msg butler.WebsocketMessage) {
			// handle received message
			ws.SendText(nextMsg.Text()) // send back what was received
		})

		ws.OnClose(func(closeMsg butler.CloseMessage) {
			// handle ws closing
		})

		return nil
	},
}
```
