package exporter

type UWSGIStats struct {
	Version           string        `json:"version"`
	ListenQueue       int           `json:"listen_queue"`
	ListenQueueErrors int           `json:"listen_queue_errors"`
	SignalQueue       int           `json:"signal_queue"`
	Load              int           `json:"load"`
	PID               int           `json:"pid"`
	UID               int           `json:"uid"`
	GID               int           `json:"gid"`
	CWD               string        `json:"cwd"`
	Sockets           []UWSGISocket `json:"sockets"`
	Workers           []UWSGIWorker `json:"workers"`
}

type UWSGISocket struct {
	Name       string `json:"name"`
	Proto      string `json:"proto"`
	Queue      int    `json:"queue"`
	MaxQueue   int    `json:"max_queue"`
	Shared     int    `json:"shared"`
	CanOffload int    `json:"can_offload"`
}

type UWSGIWorker struct {
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
	RunningTime   int         `json:"running_time"`
	LastSpawn     int64       `json:"last_spawn"`
	RespawnCount  int         `json:"respawn_count"`
	TX            int         `json:"tx"`
	AvgRt         int         `json:"avg_rt"`
	Apps          []UWSGIApp  `json:"apps"`
	Cores         []UWSGICore `json:"cores"`
}

type UWSGIApp struct {
	ID          int    `json:"id"`
	Modifier1   int    `json:"modifier1"`
	Mountpoint  string `json:"mountpoint"`
	StartupTime int    `json:"startup_time"`
	Requests    int    `json:"requests"`
	Exceptions  int    `json:"exceptions"`
	Chdir       string `json:"chdir"`
}

type UWSGICore struct {
	ID                int      `json:"id"`
	Requests          int      `json:"requests"`
	StaticRequests    int      `json:"static_requests"`
	RoutedRequests    int      `json:"routed_requests"`
	OffloadedRequests int      `json:"offloaded_requests"`
	WriteErrors       int      `json:"write_errors"`
	ReadErrors        int      `json:"read_errors"`
	InRequests        int      `json:"in_requests"`
	Vars              []string `json:"vars"`
}
