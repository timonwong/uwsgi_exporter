package collector

import (
	"encoding/json"
	"io"
)

type UwsgiSocket struct {
	Name       string `json:"name"`
	Proto      string `json:"proto"`
	Queue      int    `json:"queue"`
	MaxQueue   int    `json:"max_queue"`
	Shared     int    `json:"shared"`
	CanOffload int    `json:"can_offload"`
}

type UwsgiWorker struct {
	ID            int         `json:"id"`
	PID           int         `json:"pid"`
	Accepting     int         `json:"accepting"`
	Requests      int         `json:"requests"`
	DeltaRequests int         `json:"delta_requests"`
	Exceptions    int         `json:"exceptions"`
	HarakiriCount int         `json:"harakiri_count"`
	Signals       int         `json:"signals"`
	SignalQueue   int         `json:"signal_queue"`
	Status        string      `json:"status"`
	RSS           int         `json:"rss"`
	VSZ           int         `json:"vsz"`
	RunningTime   int64       `json:"running_time"`
	LastSpawn     int64       `json:"last_spawn"`
	RespawnCount  int         `json:"respawn_count"`
	TX            int         `json:"tx"`
	AvgRt         int         `json:"avg_rt"`
	Apps          []UwsgiApp  `json:"apps"`
	Cores         []UwsgiCore `json:"cores"`
}

type UwsgiApp struct {
	ID          int    `json:"id"`
	Modifier1   int    `json:"modifier1"`
	MountPoint  string `json:"mountpoint"`
	StartupTime int    `json:"startup_time"`
	Requests    int    `json:"requests"`
	Exceptions  int    `json:"exceptions"`
	Chdir       string `json:"chdir"`
}

type UwsgiCore struct {
	ID                int      `json:"id"`
	Requests          int      `json:"requests"`
	StaticRequests    int      `json:"static_requests"`
	RoutedRequests    int      `json:"routed_requests"`
	OffloadedRequests int      `json:"offloaded_requests"`
	WriteErrors       int      `json:"write_errors"`
	ReadErrors        int      `json:"read_errors"`
	InRequest         int      `json:"in_request"`
	Vars              []string `json:"vars"`
}

type UwsgiCache struct {
	Hits           int    `json:"hits"`
	Misses         int    `json:"miss"`
	Items          int    `json:"items"`
	MaxItems       int    `json:"max_items"`
	Full           int    `json:"full"`
	Hash           string `json:"hash"`
	HashSize       int    `json:"hashsize"`
	KeySize        int    `json:"keysize"`
	Blocks         int    `json:"blocks"`
	BlockSize      int    `json:"blocksize"`
	LastModifiedAt int    `json:"last_modified_at"`
	Name           string `json:"name"`
}

type UwsgiStats struct {
	Version           string        `json:"version"`
	ListenQueue       int           `json:"listen_queue"`
	ListenQueueErrors int           `json:"listen_queue_errors"`
	SignalQueue       int           `json:"signal_queue"`
	Load              int           `json:"load"`
	PID               int           `json:"pid"`
	UID               int           `json:"uid"`
	GID               int           `json:"gid"`
	CWD               string        `json:"cwd"`
	Sockets           []UwsgiSocket `json:"sockets"`
	Workers           []UwsgiWorker `json:"workers"`
	Caches            []UwsgiCache  `json:"caches"`
}

func parseUwsgiStatsFromIO(r io.Reader) (*UwsgiStats, error) {
	var uwsgiStats UwsgiStats
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&uwsgiStats)
	if err != nil {
		return nil, err
	}
	return &uwsgiStats, nil
}
