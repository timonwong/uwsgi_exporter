package exporter

import (
	"testing"
	"time"

	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net"
)

func TestUnixStatsReader(t *testing.T) {
	a := assert.New(t)

	content, err := ioutil.ReadFile("../testdata/sample.json")
	a.NoError(err)

	ls, err := newLocalServer("unix")
	a.NoError(err)

	defer ls.teardown()
	ch := make(chan error, 1)

	handler := func(ls *localServer, ln net.Listener) {
		func(ln net.Listener, ch chan<- error) {
			defer close(ch)

			switch ln := ln.(type) {
			case *net.UnixListener:
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
			c.SetReadDeadline(time.Now().Add(someTimeout))
			c.SetWriteDeadline(time.Now().Add(someTimeout))

			if _, err := c.Write(content); err != nil {
				ch <- err
				return
			}
		}(ln, ch)
	}

	ls.buildup(handler)

	uri := "unix://" + ls.Listener.Addr().String()
	duration := 5 * time.Second
	reader, err := NewStatsReader(uri, duration)
	a.NoError(err)

	uwsgiStats, err := reader.Read()
	a.NoError(err)

	a.Equal(uwsgiStats.Version, "2.0.12")
}
