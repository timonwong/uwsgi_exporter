package exporter

import (
	"net"
	"net/url"
	"time"

	"github.com/prometheus/common/log"
)

type tcpStatsReader struct {
	host    string
	timeout time.Duration
}

func init() {
	StatsReaderCreators = append(StatsReaderCreators, newTCPStatsReader)
}

func newTCPStatsReader(u *url.URL, uri string, timeout time.Duration) StatsReader {
	if u.Scheme != "tcp" {
		return nil
	}

	return &tcpStatsReader{
		host:    u.Host,
		timeout: timeout,
	}
}

func (reader *tcpStatsReader) Read() (*UwsgiStats, error) {
	conn, err := net.Dial("tcp", reader.host)
	if err != nil {
		log.Errorf("Error while reading uwsgi stats from tcp %s: %s", reader.host, err)
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
