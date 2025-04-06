package proxy

import (
	"bytes"
	"io"
	"net"
	"strings"
	"time"

	"github.com/fatedier/frp/utils/log"
)

const (
	detectBytesLen = 512
	maxCheckDelay  = 500 * time.Millisecond
)

var httpMethods = []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"}

type ProtocolFilter struct {
	logger log.Logger
	rules  map[string]bool // 动态规则存储
}

func NewProtocolFilter() *ProtocolFilter {
	return &ProtocolFilter{
		rules: map[string]bool{
			"http":  true,
			"https": true,
		},
		logger: log.NewPrefixLogger("[protocol_filter]"),
	}
}

// 动态更新检测规则（可通过API调用）
func (pf *ProtocolFilter) UpdateRules(newRules map[string]bool) {
	pf.rules = newRules
	pf.logger.Info("Protocol filter rules updated: %+v", newRules)
}

func (pf *ProtocolFilter) CheckAndClose(conn net.Conn) bool {
	if pf.Check(conn) {
		conn.Close()
		return true
	}
	return false
}

func (pf *ProtocolFilter) Check(conn net.Conn) bool {
	// 设置检测超时
	_ = conn.SetReadDeadline(time.Now().Add(maxCheckDelay))
	defer conn.SetReadDeadline(time.Time{})

	// 嗅探前512字节
	buf := make([]byte, detectBytesLen)
	n, _ := io.ReadFull(conn, buf)
	sniffData := buf[:n]

	// 多维度检测
	isHTTP := pf.checkHTTP(sniffData) ||
		pf.checkTLS(sniffData) ||
		pf.checkHTTPHeaders(sniffData)

	// 重置读取位置
	conn = NewBufferedConn(conn, sniffData)
	return isHTTP
}

func (pf *ProtocolFilter) checkHTTP(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// 检测HTTP方法
	firstLine := bytes.SplitN(data, []byte{'\r', '\n'}, 2)[0]
	for _, method := range httpMethods {
		if bytes.HasPrefix(firstLine, []byte(method+" ")) {
			pf.logger.Warn("HTTP method detected: %s", method)
			return true
		}
	}
	return false
}

func (pf *ProtocolFilter) checkTLS(data []byte) bool {
	// TLS握手特征检测
	if len(data) < 3 {
		return false
	}
	return data[0] == 0x16 && data[1] == 0x03 && data[2] <= 0x03
}

func (pf *ProtocolFilter) checkHTTPHeaders(data []byte) bool {
	// 检测典型HTTP头
	headers := []string{"Host:", "User-Agent:", "Content-Type:", "Accept:"}
	lowerData := bytes.ToLower(data)
	for _, header := range headers {
		if bytes.Contains(lowerData, bytes.ToLower([]byte(header))) {
			pf.logger.Warn("HTTP header detected: %s", header)
			return true
		}
	}
	return false
}

// 带缓冲的连接包装器
type BufferedConn struct {
	net.Conn
	buf *bytes.Reader
}

func NewBufferedConn(c net.Conn, buf []byte) *BufferedConn {
	return &BufferedConn{
		Conn: c,
		buf:  bytes.NewReader(buf),
	}
}

func (bc *BufferedConn) Read(p []byte) (n int, err error) {
	if bc.buf.Len() > 0 {
		return bc.buf.Read(p)
	}
	return bc.Conn.Read(p)
}

func isHTTPRequest(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	methods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"}
	firstLine := string(bytes.SplitN(data, []byte{'\r', '\n'}, 2)[0])
	for _, method := range methods {
		if strings.HasPrefix(firstLine, method+" ") {
			return true
		}
	}
	return false
}

func isHTTPSRequest(data []byte) bool {
	// TLS handshake starts with 0x16 (Content Type: Handshake)
	// followed by 0x03 (SSL/TLS version major)
	// and 0x01 to 0x03 (SSL/TLS version minor)
	if len(data) >= 3 && data[0] == 0x16 && data[1] == 0x03 && data[2] >= 0x01 && data[2] <= 0x03 {
		return true
	}
	return false
}
