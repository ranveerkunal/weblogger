package weblogger

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	HeartBeatTimeout = 30 * time.Second
)

type Logger struct {
	w map[string]chan<- string
}

func NewLogger() *Logger {
	return &Logger{w: map[string]chan<- string{}}
}

func (l *Logger) Log(r *http.Request, w http.ResponseWriter) {
	sock, err := WebSocket(r, w)
	if err != nil {
		http.Error(w, "Websocket Failed"+err.Error(), http.StatusInternalServerError)
		return
	}
	l.w[sock.Addr.String()] = sock.W
	last_activity := time.Now()
	for {
		select {
		case <-sock.R:
			last_activity = time.Now()
		case <-sock.RE:
			delete(l.w, sock.Addr.String())
			return
		case <-time.After(HeartBeatTimeout):
			if last_activity.Add(HeartBeatTimeout).Before(time.Now()) {
				delete(l.w, sock.Addr.String())
				close(sock.W)
				return
			}
		}
	}
}

type Message struct {
	MS     int64       `json:"ts,omitempty"`
	Text   string      `json:"text,omitempty"`
	Binary interface{} `json:"binary,omitempty"`
}

func (l *Logger) Remotef(format string, args ...interface{}) {
	for _, w := range l.w {
		msg := &Message{
			MS:   time.Now().UnixNano() / int64(time.Millisecond),
			Text: fmt.Sprintf(format, args...),
		}
		data, _ := json.Marshal(msg)
		str := base64.StdEncoding.EncodeToString(data)
		w <- str
	}
}
