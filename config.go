package centrifuge

const (
	DefaultPrivateChannelPrefix = "$"
	DefaultTimeoutMilliseconds  = 5000
	DefaultPingMilliseconds     = 25000
	DefaultPongMilliseconds     = 10000
	DefaultWebsocketCompression = true
	// NumReconnect is maximum number of reconnect attempts, 0 means reconnect forever.
	DefaultBackoffNumReconnect = 10
	// MinMilliseconds is a minimum value of the reconnect interval.
	DefaultBackoffMinMilliseconds = 5 * 1000
	// MaxMilliseconds is a maximum value of the reconnect interval.
	DefaultBackoffMaxMilliseconds = 20 * 1000
	// Factor is the multiplying factor for each increment step.
	DefaultBackoffFactor = 2
	// Jitter eases contention by randomizing backoff steps.
	DefaultBackoffJitter = true
)

// WsConfig contains various client options.
type Config struct {
	TimeoutMilliseconds  int
	PrivateChannelPrefix string
	WebsocketCompression bool
	Ping                 bool
	PingMilliseconds     int
	PongMilliseconds     int
	NumReconnect         int
	Factor               float64
	Jitter               bool
	MinMilliseconds      int
	MaxMilliseconds      int
}

// DefaultConfig returns WsConfig with default options.
func DefaultConfig() *Config {
	return &Config{
		NumReconnect:         DefaultBackoffNumReconnect,
		Factor:               DefaultBackoffFactor,
		Jitter:               DefaultBackoffJitter,
		MinMilliseconds:      DefaultBackoffMinMilliseconds,
		MaxMilliseconds:      DefaultBackoffMaxMilliseconds,
		PingMilliseconds:     DefaultPingMilliseconds,
		PongMilliseconds:     DefaultPongMilliseconds,
		PrivateChannelPrefix: DefaultPrivateChannelPrefix,
		TimeoutMilliseconds:  DefaultTimeoutMilliseconds,
		WebsocketCompression: DefaultWebsocketCompression,
	}
}
