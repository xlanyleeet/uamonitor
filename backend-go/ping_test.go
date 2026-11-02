package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/netip"
	"testing"
	"time"
)

func TestAddressPolicy(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1", "0.0.0.0", "192.0.2.1", "198.18.0.1", "224.0.0.1", "::1", "::ffff:127.0.0.1", "fc00::1", "fe80::1", "64:ff9b::a00:1", "2002:7f00:1::"} {
		if publicAddress(netip.MustParseAddr(raw)) {
			t.Error("unsafe allowed", raw)
		}
	}
	for _, raw := range []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"} {
		if !publicAddress(netip.MustParseAddr(raw)) {
			t.Error("public blocked", raw)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := Ping(ctx, "127.0.0.1", 25566, false); !errors.Is(err, ErrForbidden) {
		t.Fatal(err)
	}
}

func serveFixture(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	n, err := readVarInt(conn)
	if err != nil {
		return
	}
	if _, err = io.CopyN(io.Discard, conn, int64(n)); err != nil {
		return
	}
	n, err = readVarInt(conn)
	if err != nil {
		return
	}
	if _, err = io.CopyN(io.Discard, conn, int64(n)); err != nil {
		return
	}
	data := []byte(`{"version":{"name":"Test 1.21"},"players":{"online":7,"max":50},"favicon":"invalid"}`)
	body := appendVarInt([]byte{0}, len(data))
	body = append(body, data...)
	conn.Write(append(appendVarInt(nil, len(body)), body...))
}

func TestMinecraftStatus(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	go serveFixture(server)
	client.SetDeadline(time.Now().Add(2 * time.Second))
	r, err := queryStatus(client, "example.org", 25565)
	if err != nil || r.Online != 7 || r.Max != 50 || r.Version != "Test 1.21" || r.Icon != "" {
		t.Fatal(r, err)
	}
}

func TestVarIntBounds(t *testing.T) {
	for _, n := range []int{0, 127, 128, 25565, 2147483647} {
		v, err := readVarInt(bytes.NewReader(appendVarInt(nil, n)))
		if v != n || err != nil {
			t.Fatal(n, v, err)
		}
	}
	for _, b := range [][]byte{{128, 128, 128, 128, 128, 0}, {255, 255, 255, 255, 127}, {128}, {255, 255, 255, 255, 15}} {
		if _, err := readVarInt(bytes.NewReader(b)); err == nil {
			t.Fatal("malformed varint accepted")
		}
	}
}

func TestOversizedPacketRejected(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	go func() {
		defer server.Close()
		n, _ := readVarInt(server)
		io.CopyN(io.Discard, server, int64(n))
		n, _ = readVarInt(server)
		io.CopyN(io.Discard, server, int64(n))
		server.Write(appendVarInt(nil, (1<<20)+1))
	}()
	client.SetDeadline(time.Now().Add(time.Second))
	if _, err := queryStatus(client, "example.org", 25565); err == nil {
		t.Fatal("oversized packet accepted")
	}
}

func TestInputValidation(t *testing.T) {
	for _, host := range []string{"localhost", "mc.example.org", "2001:4860:4860::8888"} {
		if !validHost(host) {
			t.Fatal(host)
		}
	}
	for _, host := range []string{"", "a/b", "-host", "host..org", "host:25565"} {
		if validHost(host) {
			t.Fatal(host)
		}
	}
	for _, url := range []string{"javascript:alert(1)", "data:text/html,x", "https://user:pass@example.org"} {
		if safeURL(url) {
			t.Fatal(url)
		}
	}
	if !safeURL("https://example.org") || validIcon("data:image/svg+xml;base64,PHN2Zz4=") {
		t.Fatal("validation failed")
	}
}
