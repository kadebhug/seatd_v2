package realtime

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	wsGUID       = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	opText       = 1
	opClose      = 8
	opPing       = 9
	opPong       = 10
	maxFrameSize = 1 << 20
)

type WebSocket struct {
	conn      net.Conn
	reader    *bufio.Reader
	writeMu   sync.Mutex
	readLimit int64
}

func Upgrade(w http.ResponseWriter, r *http.Request, readLimit int64) (*WebSocket, error) {
	if !headerHasToken(r.Header, "Connection", "upgrade") || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "upgrade required", http.StatusUpgradeRequired)
		return nil, errors.New("request is not a websocket upgrade")
	}
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		http.Error(w, "unsupported websocket version", http.StatusBadRequest)
		return nil, errors.New("unsupported websocket version")
	}
	key := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		http.Error(w, "websocket key required", http.StatusBadRequest)
		return nil, errors.New("websocket key required")
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket unsupported", http.StatusInternalServerError)
		return nil, errors.New("response writer does not support hijacking")
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("hijacking connection: %w", err)
	}
	accept := websocketAccept(key)
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err := rw.WriteString(response); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("writing upgrade response: %w", err)
	}
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("flushing upgrade response: %w", err)
	}
	return &WebSocket{conn: conn, reader: rw.Reader, readLimit: readLimit}, nil
}

func (ws *WebSocket) Close() error {
	return ws.conn.Close()
}

func (ws *WebSocket) ReadLoop(ctx context.Context) error {
	for {
		_ = ws.conn.SetReadDeadline(time.Now().Add(75 * time.Second))
		op, payload, err := ws.readFrame()
		if err != nil {
			return err
		}
		switch op {
		case opClose:
			_ = ws.writeFrame(opClose, nil)
			return io.EOF
		case opPing:
			if err := ws.writeFrame(opPong, payload); err != nil {
				return err
			}
		case opPong, opText:
		default:
			return fmt.Errorf("unsupported websocket opcode %d", op)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

func (ws *WebSocket) WriteLoop(ctx context.Context, messages <-chan Message, heartbeat time.Duration) error {
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case message, ok := <-messages:
			if !ok {
				_ = ws.writeFrame(opClose, nil)
				return nil
			}
			body, err := json.Marshal(message)
			if err != nil {
				return fmt.Errorf("encoding realtime message: %w", err)
			}
			if err := ws.writeFrame(opText, body); err != nil {
				return err
			}
		case <-ticker.C:
			if err := ws.writeFrame(opPing, nil); err != nil {
				return err
			}
		}
	}
}

func (ws *WebSocket) readFrame() (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(ws.reader, header); err != nil {
		return 0, nil, err
	}
	fin := header[0]&0x80 != 0
	if !fin {
		return 0, nil, errors.New("fragmented websocket frames are not supported")
	}
	op := header[0] & 0x0f
	masked := header[1]&0x80 != 0
	if !masked {
		return 0, nil, errors.New("client websocket frame is not masked")
	}
	length := uint64(header[1] & 0x7f)
	switch length {
	case 126:
		var extended [2]byte
		if _, err := io.ReadFull(ws.reader, extended[:]); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(extended[:]))
	case 127:
		var extended [8]byte
		if _, err := io.ReadFull(ws.reader, extended[:]); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(extended[:])
	}
	if length > uint64(ws.readLimit) || length > maxFrameSize {
		return 0, nil, errors.New("websocket frame exceeds read limit")
	}
	var mask [4]byte
	if _, err := io.ReadFull(ws.reader, mask[:]); err != nil {
		return 0, nil, err
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(ws.reader, payload); err != nil {
		return 0, nil, err
	}
	for i := range payload {
		payload[i] ^= mask[i%4]
	}
	return op, payload, nil
}

func (ws *WebSocket) writeFrame(op byte, payload []byte) error {
	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()
	_ = ws.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	header := []byte{0x80 | op}
	length := len(payload)
	switch {
	case length < 126:
		header = append(header, byte(length))
	case length <= 65535:
		header = append(header, 126, 0, 0)
		binary.BigEndian.PutUint16(header[len(header)-2:], uint16(length))
	default:
		header = append(header, 127, 0, 0, 0, 0, 0, 0, 0, 0)
		binary.BigEndian.PutUint64(header[len(header)-8:], uint64(length))
	}
	if _, err := ws.conn.Write(header); err != nil {
		return fmt.Errorf("writing websocket frame header: %w", err)
	}
	if len(payload) == 0 {
		return nil
	}
	if _, err := ws.conn.Write(payload); err != nil {
		return fmt.Errorf("writing websocket frame payload: %w", err)
	}
	return nil
}

func websocketAccept(key string) string {
	sum := sha1.Sum([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func headerHasToken(header http.Header, key, token string) bool {
	for _, value := range header.Values(key) {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), token) {
				return true
			}
		}
	}
	return false
}
