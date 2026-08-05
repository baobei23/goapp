package aerospike

import "testing"

func TestConfigPortDefaults(t *testing.T) {
	cases := map[string]struct {
		in   string
		want int
	}{
		"empty":   {"", defaultPort},
		"valid":   {"3100", 3100},
		"invalid": {"abc", defaultPort},
		"zero":    {"0", defaultPort},
	}
	for name, c := range cases {
		cfg := &Config{Port: c.in}
		if got := cfg.port(); got != c.want {
			t.Errorf("%s: port() = %d, want %d", name, got, c.want)
		}
	}
}

func TestClientPolicyMapping(t *testing.T) {
	cfg := &Config{Username: "admin", Password: "secret", ConnQueueSize: 10}
	p := cfg.clientPolicy()

	if p.User != "admin" || p.Password != "secret" {
		t.Errorf("credentials not mapped: user=%q pass=%q", p.User, p.Password)
	}
	if p.ConnectionQueueSize != 10 {
		t.Errorf("ConnectionQueueSize = %d, want 10", p.ConnectionQueueSize)
	}
	if p.Timeout != defaultDialTimeout {
		t.Errorf("Timeout = %v, want default %v", p.Timeout, defaultDialTimeout)
	}

	// No credentials -> User must stay empty.
	if u := (&Config{}).clientPolicy().User; u != "" {
		t.Errorf("expected empty user without credentials, got %q", u)
	}
}
