// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package collector

import (
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

// someTimeout is used just to test that net.Conn implementations
// don't explode when their SetFooDeadline methods are called.
// It isn't actually used for testing timeouts.
const someTimeout = 10 * time.Second

// testUnixAddr uses os.CreateTemp to get a name that is unique.
// It also uses /tmp directory in case it is prohibited to create UNIX
// sockets in TMPDIR.
func testUnixAddr() string {
	f, err := os.CreateTemp("", "uwsgi-exporter-test")
	if err != nil {
		panic(err)
	}
	addr := f.Name()
	f.Close()
	os.Remove(addr)
	return addr
}

func newLocalListener(network string) (net.Listener, error) {
	switch network {
	case "tcp":
		return net.Listen("tcp4", "127.0.0.1:0")
	case "unix":
		return net.Listen(network, testUnixAddr())
	default:
		return nil, fmt.Errorf("%s is not supported", network)
	}
}

type localServer struct {
	mu sync.RWMutex
	net.Listener

	done chan struct{} // signal that indicates server stopped
}

func (ls *localServer) buildup(handler func(*localServer, net.Listener)) {
	go func() {
		handler(ls, ls.Listener)
		close(ls.done)
	}()
}

func (ls *localServer) teardown() error {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if ls.Listener != nil {
		network := ls.Listener.Addr().Network()
		address := ls.Listener.Addr().String()
		ls.Listener.Close()
		<-ls.done
		ls.Listener = nil
		if network == "unix" {
			os.Remove(address)
		}
	}
	return nil
}

func newLocalServer(t *testing.T, network string) (*localServer, error) {
	ln, err := newLocalListener(network)
	if err != nil {
		return nil, err
	}

	s := &localServer{Listener: ln, done: make(chan struct{})}
	t.Cleanup(func() {
		s.teardown()
	})
	return s, nil
}

func justWriteHandler(content []byte, ch chan<- error) func(*localServer, net.Listener) {
	return func(ls *localServer, ln net.Listener) {
		defer close(ch)

		switch ln := ln.(type) {
		case *net.UnixListener:
			ln.SetDeadline(time.Now().Add(someTimeout))
		case *net.TCPListener:
			ln.SetDeadline(time.Now().Add(someTimeout))
		}
		c, err := ln.Accept()
		if err != nil {
			ch <- err
			return
		}
		defer c.Close()

		network := ln.Addr().Network()
		if c.LocalAddr().Network() != network || c.RemoteAddr().Network() != network {
			ch <- fmt.Errorf("got %v->%v; expected %v->%v", c.LocalAddr().Network(), c.RemoteAddr().Network(), network, network)
			return
		}

		c.SetDeadline(time.Now().Add(someTimeout))

		if _, err := c.Write(content); err != nil {
			ch <- err
			return
		}
	}
}
