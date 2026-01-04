package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	raw := flag.String("url", "", "URL like http://127.0.0.1:3000/healthz")
	timeout := flag.Duration("timeout", 2*time.Second, "timeout")
	flag.Parse()

	if *raw == "" {
		fmt.Fprintln(os.Stderr, "missing -url")
		os.Exit(2)
	}

	u, err := url.Parse(*raw)
	if err != nil || u.Scheme != "http" {
		fmt.Fprintln(os.Stderr, "only http:// supported (keeps binary small)")
		os.Exit(2)
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "80"
	}
	path := u.RequestURI()
	if path == "" {
		path = "/"
	}

	d := net.Dialer{Timeout: *timeout}
	conn, err := d.Dial("tcp", net.JoinHostPort(host, port))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(*timeout))

	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", path, u.Host)
	if _, err := conn.Write([]byte(req)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	parts := strings.SplitN(strings.TrimSpace(line), " ", 3)
	if len(parts) < 2 {
		fmt.Fprintln(os.Stderr, "bad response:", line)
		os.Exit(1)
	}
	code, err := strconv.Atoi(parts[1])
	if err != nil || code < 200 || code >= 300 {
		fmt.Fprintln(os.Stderr, "status", parts[1])
		os.Exit(1)
	}
}
