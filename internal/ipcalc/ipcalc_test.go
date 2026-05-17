package ipcalc

import (
	"net/netip"
	"testing"
)

// Single-case smoke test covering the most common subnet. Heavier
// table-driven coverage (every prefix length, /31, /32, /0, host-bits-set,
// classful boundaries) lands as we flesh out Calculate.
func TestCalculate_192_168_1_0_24(t *testing.T) {
	r := Calculate(netip.MustParsePrefix("192.168.1.0/24"))

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"Network", r.Network.String(), "192.168.1.0"},
		{"Broadcast", r.Broadcast.String(), "192.168.1.255"},
		{"HostMin", r.HostMin.String(), "192.168.1.1"},
		{"HostMax", r.HostMax.String(), "192.168.1.254"},
		{"HostCount", r.HostCount, uint64(254)},
		{"Netmask", r.Netmask.String(), "255.255.255.0"},
		{"Wildcard", r.Wildcard.String(), "0.0.0.255"},
		{"PrefixLen", r.PrefixLen, 24},
		{"Class", r.Class, "C"},
		{"IsRFC1918", r.IsRFC1918, true},
		{"HostBitsSet", r.HostBitsSet, false},
		{"BinaryAddr", r.BinaryAddr, "11000000.10101000.00000001.00000000"},
		{"BinaryMask", r.BinaryMask, "11111111.11111111.11111111.00000000"},
		{"ReverseDNS", r.ReverseDNS, "0.1.168.192.in-addr.arpa"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}
