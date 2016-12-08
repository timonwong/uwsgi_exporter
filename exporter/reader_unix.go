package exporter

import (
	"net"
	"net/url"
	"time"

	"github.com/prometheus/common/log"
)

type unixStatsReader struct {
	filename string
	timeout  time.Duration
}

func init() {
	StatsReaderCreators = append(StatsReaderCreators, newUnixStatsReader)
}

func newUnixStatsReader(u *url.URL, uri string, timeout time.Duration) StatsReader {
	if u.Scheme != "unix" {
		return nil
	}

	return &unixStatsReader{
		filename: u.Path,
		timeout:  timeout,
	}
}

func (reader *unixStatsReader) Read() (*UwsgiStats, error) {
	conn, err := net.Dial("unix", string(reader.filename))
	if err != nil {
		log.Errorf("Error while reading uwsgi stats from unix socket %s: %s", reader.filename, err)
		return nil, err
	}
	defer conn.Close()

	err = conn.SetDeadline(time.Now().Add(reader.timeout))
	if err != nil {
		log.Errorf("Failed to set deadline: %s", err)
		return nil, err
	}

	uwsgiStats, err := parseUwsgiStatsFromIO(conn)
	if err != nil {
		log.Errorf("Failed to unmarshal JSON: %s", err)
		return nil, err
	}
	return uwsgiStats, nil
}
