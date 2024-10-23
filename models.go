package golog

type Config struct {
	LogStdout         bool   // print logs in stdout
	LogFile           bool   // store logs in file
	FileLocation      string // location to log file
	FileMaxSize       int    // maximum size in megabytes of the log file before it gets rotated
	FileMaxBackups    int    // maximum number of old log files to retain
	LogServer         bool   // send logs to server
	ServerApiProtocol string // api protocl `https`
	ServerApiHost     string // api host `api.example.com`
	ServerApiPort     string // api port `443`
	ServerPlatfrom    string // platform name in server
	ServerKey         string // server key
}

// ==================== //

type Enviroment int

const (
	Local       Enviroment = 0
	Development Enviroment = 1
	Testing     Enviroment = 2
	Production  Enviroment = 3
)

func (e Enviroment) String() string {

	return [...]string{"dev", "test", "prod", "local"}[e]
}

// ==================== //

type Level int

const (
	Trace Level = 0
	Debug Level = 1
	Info  Level = 2
	Warn  Level = 3
	Error Level = 4
	Fatal Level = 5
	Panic Level = 6
)

func (l Level) String() string {

	return [...]string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}[l]
}

// ==================== //

type Log struct {
	Level       Level          `json:"level"`
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	UserActions []string       `json:"user_action"`
	HttpPayload map[string]any `json:"http_payload"`
	UserPayload map[string]any `json:"user_payload"`
	Data        map[string]any `json:"data"`
	Media       map[string]any `json:"media"`
	Enviroment  Enviroment     `json:"env"`
	CodeTrace   map[string]any `json:"code_trace"`
}
