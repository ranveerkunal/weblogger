package weblogger

import (
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Socket struct {
	R    <-chan string
	RE   <-chan error
	W    chan<- string
	WE   chan<- error
	Addr net.Addr
}

func done(conn *websocket.Conn, cc int, msg string) {
	conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(cc, msg), time.Time{})
}

func read(conn *websocket.Conn, rc chan string, rec chan error) {
	defer conn.Close()
	for {
		t, p, err := conn.ReadMessage()
		if err != nil {
			rec <- err
			return
		}
		if t == websocket.TextMessage {
			rc <- string(p)
		}
	}
}

func write(conn *websocket.Conn, wc chan string, rec chan error, wec chan error) {
	defer conn.Close()
	for {
		select {
		case p, ok := <-wc:
			if !ok {
				done(conn, websocket.CloseNormalClosure, "OK")
				return
			}
			err := conn.WriteMessage(websocket.TextMessage, []byte(p))
			if err != nil {
				done(conn, websocket.CloseInternalServerErr, err.Error())
				rec <- err
				return
			}
		case err := <-wec:
			done(conn, websocket.CloseInternalServerErr, err.Error())
			return
		}
	}
}

func WebSocket(r *http.Request, w http.ResponseWriter) (*Socket, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	// Make channels.
	rc := make(chan string)
	rec := make(chan error)
	wc := make(chan string)
	wec := make(chan error)
	cs := &Socket{rc, rec, wc, wec, conn.RemoteAddr()}

	// Start read and write routines.
	go read(conn, rc, rec)
	go write(conn, wc, rec, wec)
	return cs, nil
}
