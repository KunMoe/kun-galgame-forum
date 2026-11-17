package agent

import "testing"

func TestVisitorIPIsTrustedOnlyFromAPrivatePeer(t *testing.T) {
	for _, c := range []struct{ peer, forwarded, want string }{
		{"10.0.1.4", "203.0.113.9", "203.0.113.9"},
		{"127.0.0.1", " 2001:db8::1 ", "2001:db8::1"},
		{"198.51.100.2", "203.0.113.9", "198.51.100.2"},
		{"10.0.1.4", "", "10.0.1.4"},
		{"10.0.1.4", "203.0.113.9, 10.0.0.1", "10.0.1.4"},
	} {
		if got := visitorIP(c.peer, c.forwarded); got != c.want {
			t.Errorf("visitorIP(%q, %q) = %q, want %q", c.peer, c.forwarded, got, c.want)
		}
	}
}
