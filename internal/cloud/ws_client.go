package cloud

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

// WSClientConn 轻量级纯标准库客户端 WebSocket 连接封装
type WSClientConn struct {
	conn    net.Conn
	reader  *bufio.Reader
	writeMu sync.Mutex
	closed  bool
}

// DialWebSocket 建立出站 WebSocket 客户端长连接
func DialWebSocket(rawURL string, secret string) (*WSClientConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	host := u.Host
	if !strings.Contains(host, ":") {
		if u.Scheme == "wss" || u.Scheme == "https" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	var conn net.Conn
	if u.Scheme == "wss" || u.Scheme == "https" {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // 允许自建自签名证书 VPS
			ServerName:         strings.Split(u.Host, ":")[0],
		}
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", host, tlsConfig)
	} else {
		conn, err = net.DialTimeout("tcp", host, 10*time.Second)
	}
	if err != nil {
		return nil, err
	}

	// 生成随机 Sec-WebSocket-Key
	nonce := make([]byte, 16)
	_, _ = rand.Read(nonce)
	secKey := base64.StdEncoding.EncodeToString(nonce)

	reqPath := u.Path
	if reqPath == "" {
		reqPath = "/ws/router"
	}
	if u.RawQuery != "" {
		reqPath += "?" + u.RawQuery
	}

	req := fmt.Sprintf("GET %s HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Upgrade: websocket\r\n"+
		"Connection: Upgrade\r\n"+
		"Sec-WebSocket-Key: %s\r\n"+
		"Sec-WebSocket-Version: 13\r\n"+
		"X-Router-Secret: %s\r\n\r\n",
		reqPath, u.Host, secKey, secret)

	if _, err := conn.Write([]byte(req)); err != nil {
		_ = conn.Close()
		return nil, err
	}

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	if !strings.Contains(statusLine, "101") {
		_ = conn.Close()
		return nil, fmt.Errorf("websocket handshake failed: %s", strings.TrimSpace(statusLine))
	}

	// 消费握手 Headers 直到空行
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	return &WSClientConn{
		conn:   conn,
		reader: reader,
	}, nil
}

// ReadMessage 读取文本帧
func (ws *WSClientConn) ReadMessage() (string, error) {
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
		case 0x8: // Close
			_ = ws.Close()
			return "", io.EOF
		case 0x9: // Ping
			_ = ws.writeFrame(0xA, payload) // Reply Pong
		case 0x1: // Text
			if fin {
				return string(payload), nil
			}
		}
	}
}

// WriteMessage 发送客户端掩码文本消息
func (ws *WSClientConn) WriteMessage(text string) error {
	return ws.writeFrame(0x1, []byte(text))
}

// WritePing 发送心跳 Ping
func (ws *WSClientConn) WritePing() error {
	return ws.writeFrame(0x9, []byte("ping"))
}

// writeFrame 发送带 Mask 掩码的客户端 WebSocket 帧 (RFC 6455 Client Requirement)
func (ws *WSClientConn) writeFrame(opcode byte, payload []byte) error {
	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()

	if ws.closed {
		return errors.New("connection closed")
	}

	length := len(payload)
	var header []byte

	// 客户端发送帧必须带 Mask bit (0x80)
	if length <= 125 {
		header = []byte{0x80 | opcode, 0x80 | byte(length)}
	} else if length <= 65535 {
		header = make([]byte, 4)
		header[0] = 0x80 | opcode
		header[1] = 0x80 | 126
		binary.BigEndian.PutUint16(header[2:], uint16(length))
	} else {
		header = make([]byte, 10)
		header[0] = 0x80 | opcode
		header[1] = 0x80 | 127
		binary.BigEndian.PutUint64(header[2:], uint64(length))
	}

	maskKey := make([]byte, 4)
	_, _ = rand.Read(maskKey)

	maskedPayload := make([]byte, length)
	for i := 0; i < length; i++ {
		maskedPayload[i] = payload[i] ^ maskKey[i%4]
	}

	frame := append(header, maskKey...)
	frame = append(frame, maskedPayload...)

	if err := ws.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}

	if _, err := ws.conn.Write(frame); err != nil {
		ws.closed = true
		return err
	}

	return nil
}

// Close 关闭连接
func (ws *WSClientConn) Close() error {
	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()

	if ws.closed {
		return nil
	}
	ws.closed = true
	return ws.conn.Close()
}
