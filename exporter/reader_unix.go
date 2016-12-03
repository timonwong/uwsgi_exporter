package exporter

import (
	"net"
	"net/url"
	"time"

	"github.com/prometheus/common/log"
)

// UnixStatsReader reads uwsgi stats from specified unix socket.
type UnixStatsReader struct {
	filename string
}

func init() {
	StatsReaderCreators = append(StatsReaderCreators, newUnixStatsReader)
}

func newUnixStatsReader(u *url.URL, uri string, timeout time.Duration) StatsReader {
	if u.Scheme != "unix" {
		return nil
	}

	return &UnixStatsReader{
		filename: u.Path,
	}
}

func (reader *UnixStatsReader) Read() (*UwsgiStats, error) {
	conn, err := net.Dial("unix", string(reader.filename))
	if err != nil {
		log.Errorf("Error while reading uwsgi stats from unix socket: %s", reader.filename)
		return nil, err
	}
	defer conn.Close()

	uwsgiStats, err := parseUwsgiStatsFromIO(conn)
	if err != nil {
		log.Errorf("Failed to unmarshal JSON: %s", err)
		return nil, err
	}
	return uwsgiStats, nil
}
