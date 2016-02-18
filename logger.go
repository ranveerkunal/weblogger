package weblogger

import (
	"net/http"
	"os"
	"time"
)

const (
	HeartBeatTimeout = 30 * time.Second
)

type RemoteWriter struct {
	w map[string]chan<- []byte
}

func NewWriter() *RemoteWriter {
	return &RemoteWriter{w: map[string]chan<- []byte{}}
}

func (w *RemoteWriter) Handler(rw http.ResponseWriter, r *http.Request) {
	sock, err := WebSocket(r, rw)
	if err != nil {
		http.Error(rw, "Websocket Failed"+err.Error(), http.StatusInternalServerError)
		return
	}
	w.w[sock.Addr.String()] = sock.W
	last_activity := time.Now()
	for {
		select {
		case <-sock.R:
			last_activity = time.Now()
		case <-sock.RE:
			delete(w.w, sock.Addr.String())
			return
		case <-time.After(HeartBeatTimeout):
			if last_activity.Add(HeartBeatTimeout).Before(time.Now()) {
				delete(w.w, sock.Addr.String())
				close(sock.W)
				return
			}
		}
	}
}

func (w *RemoteWriter) Write(p []byte) (n int, err error) {
	for _, w := range w.w {
		go func() { w <- p }()
	}
	return os.Stderr.Write(p)
}
