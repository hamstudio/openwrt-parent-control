package relay

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	opContinuation = 0x0
	opText         = 0x1
	opBinary       = 0x2
	opClose        = 0x8
	opPing         = 0x9
	opPong         = 0xA
)

// WSConn 轻量级纯标准库 WebSocket 连接封装 (RFC 6455)
type WSConn struct {
	conn   net.Conn
	reader *bufio.Reader
	writeMu sync.Mutex
	closed bool
}

// Upgrade 将标准 HTTP 请求升级为 WebSocket 连接
func Upgrade(w http.ResponseWriter, r *http.Request) (*WSConn, error) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return nil, errors.New("websocket: method not allowed")
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return nil, errors.New("websocket: missing Sec-WebSocket-Key")
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Webserver doesn't support hijacking", http.StatusInternalServerError)
		return nil, errors.New("websocket: hijacking not supported")
	}

	conn, buf, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("hijack failed: %w", err)
	}

	// 计算 Sec-WebSocket-Accept
	h := sha1.New()
	h.Write([]byte(key + wsGUID))
	acceptKey := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// 发送 101 Switching Protocols 响应
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"

	if _, err := conn.Write([]byte(resp)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &WSConn{
		conn:   conn,
		reader: buf.Reader,
	}, nil
}

// ReadMessage 读取下一条文本消息
func (ws *WSConn) ReadMessage() (string, error) {
	for {
		header := make([]byte, 2)
		if _, err := io.ReadFull(ws.reader, header); err != nil {
			return "", err
		}

		fin := (header[0] & 0x80) != 0
		opcode := header[0] & 0x0F
		masked := (header[1] & 0x80) != 0
		payloadLen := uint64(header[1] & 0x7F)

		if payloadLen == 126 {
			extLen := make([]byte, 2)
			if _, err := io.ReadFull(ws.reader, extLen); err != nil {
				return "", err
			}
			payloadLen = uint64(binary.BigEndian.Uint16(extLen))
		} else if payloadLen == 127 {
			extLen := make([]byte, 8)
			if _, err := io.ReadFull(ws.reader, extLen); err != nil {
				return "", err
			}
			payloadLen = binary.BigEndian.Uint64(extLen)
		}

		var maskKey []byte
		if masked {
			maskKey = make([]byte, 4)
			if _, err := io.ReadFull(ws.reader, maskKey); err != nil {
				return "", err
			}
		}

		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(ws.reader, payload); err != nil {
			return "", err
		}

		if masked {
			for i := uint64(0); i < payloadLen; i++ {
				payload[i] ^= maskKey[i%4]
			}
		}

		switch opcode {
		case opClose:
			_ = ws.Close()
			return "", io.EOF
		case opPing:
			_ = ws.writeFrame(opPong, payload)
		case opPong:
			// ignore pong
		case opText:
			if fin {
				return string(payload), nil
			}
		default:
			// 忽略其他帧类型
		}
	}
}

// WriteMessage 发送文本消息
func (ws *WSConn) WriteMessage(text string) error {
	return ws.writeFrame(opText, []byte(text))
}

// WritePing 发送心跳 Ping
func (ws *WSConn) WritePing() error {
	return ws.writeFrame(opPing, []byte("ping"))
}

// writeFrame 构造并发送 RFC 6455 帧
func (ws *WSConn) writeFrame(opcode byte, payload []byte) error {
	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()

	if ws.closed {
		return errors.New("connection closed")
	}

	length := len(payload)
	var header []byte

	if length <= 125 {
		header = []byte{0x80 | opcode, byte(length)}
	} else if length <= 65535 {
		header = make([]byte, 4)
		header[0] = 0x80 | opcode
		header[1] = 126
		binary.BigEndian.PutUint16(header[2:], uint16(length))
	} else {
		header = make([]byte, 10)
		header[0] = 0x80 | opcode
		header[1] = 127
		binary.BigEndian.PutUint64(header[2:], uint64(length))
	}

	if err := ws.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}

	if _, err := ws.conn.Write(append(header, payload...)); err != nil {
		ws.closed = true
		return err
	}

	return nil
}

// Close 关闭底层连接
func (ws *WSConn) Close() error {
	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()

	if ws.closed {
		return nil
	}
	ws.closed = true
	_ = ws.writeFrame(opClose, []byte{})
	return ws.conn.Close()
}
