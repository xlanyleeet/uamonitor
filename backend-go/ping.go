package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrForbidden = errors.New("target address is forbidden")

type PingResult struct {
	Online, Max   int
	Version, Icon string
}

func safeURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Hostname() != "" && u.User == nil
}

func validHost(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}

var forbiddenPrefixes = func() []netip.Prefix {
	values := []string{"0.0.0.0/8", "100.64.0.0/10", "169.254.0.0/16", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "::/96", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64", "2001::/23", "2002::/16", "3fff::/20"}
	out := make([]netip.Prefix, 0, len(values))
	for _, v := range values {
		out = append(out, netip.MustParsePrefix(v))
	}
	return out
}()

func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	for _, p := range forbiddenPrefixes {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}

func validIcon(icon string) bool {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(icon, prefix) || len(icon) > 65536 {
		return false
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(icon, prefix))
	return err == nil && len(data) >= 8 && bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10})
}

func Ping(ctx context.Context, host string, port int, allowPrivate bool) (PingResult, error) {
	if !validHost(host) || port < 1 || port > 65535 {
		return PingResult{}, fmt.Errorf("%w: invalid hostname or port", ErrForbidden)
	}
	dialHost, dialPort := host, port
	if port == 25565 && net.ParseIP(host) == nil {
		_, records, err := net.DefaultResolver.LookupSRV(ctx, "minecraft", "tcp", host)
		if err == nil && len(records) > 0 {
			dialHost = strings.TrimSuffix(records[0].Target, ".")
			dialPort = int(records[0].Port)
		}
	}
	if !validHost(dialHost) || dialPort < 1 {
		return PingResult{}, ErrForbidden
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", dialHost)
	if err != nil {
		return PingResult{}, err
	}
	if len(ips) == 0 {
		return PingResult{}, errors.New("no target addresses")
	}
	// Validate ALL answers, then connect to the numeric address. No second DNS lookup/rebinding.
	for _, ip := range ips {
		if !allowPrivate && !publicAddress(ip) {
			return PingResult{}, ErrForbidden
		}
	}
	var lastErr error
	for _, ip := range ips {
		conn, e := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), strconv.Itoa(dialPort)))
		if e != nil {
			lastErr = e
			continue
		}
		result, e := func() (PingResult, error) {
			defer conn.Close()
			deadline := time.Now().Add(5 * time.Second)
			if d, ok := ctx.Deadline(); ok {
				deadline = d
			}
			if e := conn.SetDeadline(deadline); e != nil {
				return PingResult{}, e
			}
			stop := context.AfterFunc(ctx, func() { conn.Close() })
			defer stop()
			return queryStatus(conn, host, port)
		}()
		if e == nil {
			return result, nil
		}
		lastErr = e
	}
	return PingResult{}, lastErr
}

func appendVarInt(b []byte, value int) []byte {
	v := uint32(value)
	for {
		x := byte(v & 127)
		v >>= 7
		if v != 0 {
			x |= 128
		}
		b = append(b, x)
		if v == 0 {
			return b
		}
	}
}

func readVarInt(r io.Reader) (int, error) {
	var out uint32
	for i := 0; i < 5; i++ {
		var b [1]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, err
		}
		if i == 4 && b[0] > 15 {
			return 0, errors.New("varint overflow")
		}
		out |= uint32(b[0]&127) << uint(7*i)
		if b[0]&128 == 0 {
			if out > 2147483647 {
				return 0, errors.New("negative varint")
			}
			return int(out), nil
		}
	}
	return 0, errors.New("varint too long")
}

func queryStatus(conn net.Conn, host string, port int) (PingResult, error) {
	handshake := appendVarInt(nil, 0)
	handshake = appendVarInt(handshake, 767)
	handshake = appendVarInt(handshake, len(host))
	handshake = append(handshake, host...)
	handshake = binary.BigEndian.AppendUint16(handshake, uint16(port))
	handshake = appendVarInt(handshake, 1)
	packet := append(appendVarInt(nil, len(handshake)), handshake...)
	packet = append(packet, 1, 0) // status request: length=1, packet ID=0
	if _, err := io.Copy(conn, bytes.NewReader(packet)); err != nil {
		return PingResult{}, err
	}
	length, err := readVarInt(conn)
	if err != nil {
		return PingResult{}, err
	}
	if length < 2 || length > 1<<20 {
		return PingResult{}, errors.New("invalid status packet length")
	}
	data := make([]byte, length)
	if _, err = io.ReadFull(conn, data); err != nil {
		return PingResult{}, err
	}
	r := bytes.NewReader(data)
	id, err := readVarInt(r)
	if err != nil || id != 0 {
		return PingResult{}, errors.New("invalid status packet ID")
	}
	n, err := readVarInt(r)
	if err != nil || n != r.Len() {
		return PingResult{}, errors.New("invalid status JSON length")
	}
	var reply struct {
		Players struct {
			Online int `json:"online"`
			Max    int `json:"max"`
		} `json:"players"`
		Version struct {
			Name string `json:"name"`
		} `json:"version"`
		Favicon string `json:"favicon"`
	}
	if err = json.Unmarshal(data[len(data)-n:], &reply); err != nil {
		return PingResult{}, err
	}
	if reply.Players.Online < 0 || reply.Players.Max < 0 || reply.Players.Online > 2147483647 || reply.Players.Max > 2147483647 || len(reply.Version.Name) > 1024 {
		return PingResult{}, errors.New("invalid status values")
	}
	icon := ""
	if validIcon(reply.Favicon) {
		icon = reply.Favicon
	}
	return PingResult{reply.Players.Online, reply.Players.Max, reply.Version.Name, icon}, nil
}
