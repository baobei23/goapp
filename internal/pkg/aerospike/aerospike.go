package aerospike

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	as "github.com/aerospike/aerospike-client-go/v8"
)

const (
	defaultPort          = 3000
	defaultConnQueueSize = 256
	defaultDialTimeout   = time.Second * 3
)

// Config struct holds all the configurations required by the datastore package
type Config struct {
	Host      string `json:"host,omitempty"`
	Port      string `json:"port,omitempty"`
	Namespace string `json:"namespace,omitempty"`

	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	ConnQueueSize int           `json:"connQueueSize,omitempty"`
	DialTimeout   time.Duration `json:"dialTimeout,omitempty"`
}

// port returns the numeric port, defaulting to defaultPort when unset or invalid.
func (cfg *Config) port() int {
	p := strings.TrimSpace(cfg.Port)
	if p == "" {
		return defaultPort
	}
	n, err := strconv.Atoi(p)
	if err != nil || n <= 0 {
		return defaultPort
	}
	return n
}

// clientPolicy builds the aerospike client policy from the config.
func (cfg *Config) clientPolicy() *as.ClientPolicy {
	policy := as.NewClientPolicy()

	if cfg.DialTimeout > 0 {
		policy.Timeout = cfg.DialTimeout
	} else {
		policy.Timeout = defaultDialTimeout
	}

	if cfg.ConnQueueSize > 0 {
		policy.ConnectionQueueSize = cfg.ConnQueueSize
	} else {
		policy.ConnectionQueueSize = defaultConnQueueSize
	}

	if strings.TrimSpace(cfg.Username) != "" {
		policy.User = cfg.Username
		policy.Password = cfg.Password
	}

	// Aerospike only backs refresh tokens, so an unreachable cluster must not
	// stop the app from booting. The client is returned anyway and a background
	// tender keeps retrying; calls fail until it connects.
	policy.FailIfNotConnected = false

	return policy
}

// NewClient returns a new connected instance of the Aerospike client.
func NewClient(cfg *Config) (*as.Client, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return nil, fmt.Errorf("aerospike: host cannot be empty")
	}
	// Namespace is only used when building keys, so an empty one fails every
	// request at runtime rather than at boot. Catch it here instead.
	if strings.TrimSpace(cfg.Namespace) == "" {
		return nil, fmt.Errorf("aerospike: namespace cannot be empty")
	}

	policy := cfg.clientPolicy()

	client, err := as.NewClientWithPolicyAndHost(
		policy,
		as.NewHost(host, cfg.port()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create aerospike client: %w", err)
	}

	return client, nil
}
