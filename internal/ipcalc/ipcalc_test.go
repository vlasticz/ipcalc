package ipcalc

import (
	"net/netip"
	"testing"
)

// TestCalculate_BasicCase covers the most common subnet end-to-end —
// every Result field for 192.168.1.0/24. Other tests focus on specific
// edge cases.
func TestCalculate_BasicCase(t *testing.T) {
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
		{"IsLinkLocal", r.IsLinkLocal, false},
		{"IsMulticast", r.IsMulticast, false},
		{"IsLoopback", r.IsLoopback, false},
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

// TestCalculate_Slash31 covers RFC 3021: two host addresses, no broadcast,
// no network/broadcast distinction.
func TestCalculate_Slash31(t *testing.T) {
	r := Calculate(netip.MustParsePrefix("192.0.2.0/31"))

	if r.HostCount != 2 {
		t.Errorf("HostCount: got %d, want 2", r.HostCount)
	}
	if r.HostMin.String() != "192.0.2.0" {
		t.Errorf("HostMin: got %s, want 192.0.2.0", r.HostMin)
	}
	if r.HostMax.String() != "192.0.2.1" {
		t.Errorf("HostMax: got %s, want 192.0.2.1", r.HostMax)
	}
	if r.Broadcast.IsValid() {
		t.Errorf("Broadcast: got %s, want invalid (zero Addr)", r.Broadcast)
	}
	if r.Class != "" {
		t.Errorf("Class: got %q, want empty (not meaningfully classful at /31)", r.Class)
	}
}

// TestCalculate_Slash32 covers single-host. No broadcast, host min == max
// == network, count = 1.
func TestCalculate_Slash32(t *testing.T) {
	r := Calculate(netip.MustParsePrefix("10.20.30.40/32"))

	if r.HostCount != 1 {
		t.Errorf("HostCount: got %d, want 1", r.HostCount)
	}
	if r.Network.String() != "10.20.30.40" {
		t.Errorf("Network: got %s, want 10.20.30.40", r.Network)
	}
	if r.HostMin != r.Network || r.HostMax != r.Network {
		t.Errorf("HostMin/Max should equal Network for /32 (got min=%s max=%s)", r.HostMin, r.HostMax)
	}
	if r.Broadcast.IsValid() {
		t.Errorf("Broadcast: got %s, want invalid (zero Addr)", r.Broadcast)
	}
	if r.Class != "" {
		t.Errorf("Class: got %q, want empty (not meaningfully classful at /32)", r.Class)
	}
	if r.HostBitsSet {
		t.Errorf("HostBitsSet: want false for /32 (every bit is a network bit)")
	}
}

// TestCalculate_Slash0 covers the entire IPv4 space — host count must
// not overflow uint64 and must match 2^32 - 2.
func TestCalculate_Slash0(t *testing.T) {
	r := Calculate(netip.MustParsePrefix("0.0.0.0/0"))

	const want = uint64(1<<32) - 2 // 4_294_967_294
	if r.HostCount != want {
		t.Errorf("HostCount: got %d, want %d", r.HostCount, want)
	}
	if r.Network.String() != "0.0.0.0" {
		t.Errorf("Network: got %s, want 0.0.0.0", r.Network)
	}
	if r.Broadcast.String() != "255.255.255.255" {
		t.Errorf("Broadcast: got %s, want 255.255.255.255", r.Broadcast)
	}
	if r.HostMin.String() != "0.0.0.1" {
		t.Errorf("HostMin: got %s, want 0.0.0.1", r.HostMin)
	}
	if r.HostMax.String() != "255.255.255.254" {
		t.Errorf("HostMax: got %s, want 255.255.255.254", r.HostMax)
	}
	if r.Netmask.String() != "0.0.0.0" {
		t.Errorf("Netmask: got %s, want 0.0.0.0", r.Netmask)
	}
	if r.Wildcard.String() != "255.255.255.255" {
		t.Errorf("Wildcard: got %s, want 255.255.255.255", r.Wildcard)
	}
}

// TestCalculate_HostBitsSet: feeding a host address with a non-aligned
// prefix should normalize to the network with HostBitsSet=true. This is
// the basis of the "forgiving + warn" UX.
func TestCalculate_HostBitsSet(t *testing.T) {
	r := Calculate(netip.MustParsePrefix("192.168.1.50/24"))

	if !r.HostBitsSet {
		t.Fatalf("HostBitsSet: want true (50 != network)")
	}
	if r.Network.String() != "192.168.1.0" {
		t.Errorf("Network: got %s, want 192.168.1.0", r.Network)
	}
	if r.Normalized.String() != "192.168.1.0/24" {
		t.Errorf("Normalized: got %s, want 192.168.1.0/24", r.Normalized)
	}
	if r.Input.String() != "192.168.1.50/24" {
		t.Errorf("Input: got %s, want 192.168.1.50/24 (must preserve the original)", r.Input)
	}
}

// TestCalculate_ClassByLeadingOctet checks every classful boundary at /24.
func TestCalculate_ClassByLeadingOctet(t *testing.T) {
	cases := []struct {
		prefix string
		want   string
	}{
		{"0.0.0.0/24", "A"},     // first
		{"10.0.0.0/24", "A"},    // mid
		{"127.255.255.0/24", "A"}, // last A (127 < 128)
		{"128.0.0.0/24", "B"},   // first B
		{"191.255.255.0/24", "B"},
		{"192.0.0.0/24", "C"},
		{"223.255.255.0/24", "C"},
		{"224.0.0.0/24", "D"},
		{"239.255.255.0/24", "D"},
		{"240.0.0.0/24", "E"},
		{"255.255.255.0/24", "E"},
	}
	for _, c := range cases {
		r := Calculate(netip.MustParsePrefix(c.prefix))
		if r.Class != c.want {
			t.Errorf("%s: Class got %q, want %q", c.prefix, r.Class, c.want)
		}
	}
}

// TestCalculate_RFC1918 covers the three private ranges plus boundary
// negatives that look private but aren't.
func TestCalculate_RFC1918(t *testing.T) {
	cases := []struct {
		prefix string
		want   bool
	}{
		// Positives — all three ranges.
		{"10.0.0.0/8", true},
		{"10.255.255.0/24", true},
		{"172.16.0.0/12", true},
		{"172.20.0.0/16", true},
		{"172.31.0.0/16", true},
		{"192.168.0.0/16", true},
		{"192.168.255.0/24", true},
		// Negatives — just outside each range.
		{"9.255.255.0/24", false},
		{"11.0.0.0/24", false},
		{"172.15.0.0/16", false},
		{"172.32.0.0/16", false},
		{"192.167.255.0/24", false},
		{"192.169.0.0/24", false},
		// Far-outside.
		{"8.8.8.0/24", false},
	}
	for _, c := range cases {
		r := Calculate(netip.MustParsePrefix(c.prefix))
		if r.IsRFC1918 != c.want {
			t.Errorf("%s: IsRFC1918 got %v, want %v", c.prefix, r.IsRFC1918, c.want)
		}
	}
}

// TestCalculate_PrefixWiderThanCategory is the regression test for the bug
// where a prefix wider than the canonical private/special block was
// incorrectly classified by checking only the network address's first
// byte. The fix requires that [netU, bcastU] be wholly contained in the
// category's range.
func TestCalculate_PrefixWiderThanCategory(t *testing.T) {
	cases := []struct {
		prefix    string
		wantRFC   bool
		wantPub   bool
		wantClass string
	}{
		// User-reported regression: /13 starting at 192.168.0.0 spans
		// 192.168.0.0 - 192.175.255.255. 192.169-192.175 is public space,
		// so the prefix is NOT RFC1918 even though the network address is.
		{"192.168.0.0/13", false, true, "C"},
		// /15 of 192.168 covers 192.168 + 192.169 — second half is public.
		{"192.168.0.0/15", false, true, "C"},
		// /7 of 10.0.0.0 covers 10/8 + 11/8 — only 10/8 is RFC1918.
		{"10.0.0.0/7", false, true, "A"},
		// /11 of 172.16.0.0 covers 172.0 - 172.31 — partly RFC1918, mostly not.
		{"172.0.0.0/11", false, true, "B"},
		// Multi-class prefix: /1 of 128.0.0.0 spans B+C+D+E. No class label.
		{"128.0.0.0/1", false, true, ""},
		// Exact-block matches stay positive.
		{"192.168.0.0/16", true, false, "C"},
		{"10.0.0.0/8", true, false, "A"},
		{"172.16.0.0/12", true, false, "B"},
	}
	for _, c := range cases {
		r := Calculate(netip.MustParsePrefix(c.prefix))
		if r.IsRFC1918 != c.wantRFC {
			t.Errorf("%s: IsRFC1918 got %v, want %v", c.prefix, r.IsRFC1918, c.wantRFC)
		}
		if r.IsPublic != c.wantPub {
			t.Errorf("%s: IsPublic got %v, want %v", c.prefix, r.IsPublic, c.wantPub)
		}
		if r.Class != c.wantClass {
			t.Errorf("%s: Class got %q, want %q", c.prefix, r.Class, c.wantClass)
		}
	}
}

// TestCalculate_IsPublic covers the negation flag used to render the
// red "public" warning chip. Must be false for any RFC1918 / loopback /
// link-local / multicast address; true otherwise.
func TestCalculate_IsPublic(t *testing.T) {
	cases := []struct {
		prefix string
		want   bool
	}{
		// Private / special — IsPublic must be false.
		{"10.0.0.0/8", false},
		{"172.16.0.0/12", false},
		{"192.168.1.0/24", false},
		{"127.0.0.1/32", false},
		{"169.254.0.0/16", false},
		{"224.0.0.0/4", false},
		// Public — IsPublic must be true.
		{"8.8.8.0/24", true},
		{"1.1.1.0/24", true},
		{"203.0.113.0/24", true}, // documentation range — still warns
		{"172.32.0.0/16", true},  // just outside RFC1918
		{"192.169.0.0/16", true}, // just outside RFC1918
	}
	for _, c := range cases {
		r := Calculate(netip.MustParsePrefix(c.prefix))
		if r.IsPublic != c.want {
			t.Errorf("%s: IsPublic got %v, want %v", c.prefix, r.IsPublic, c.want)
		}
	}
}

// TestCalculate_SpecialRanges covers link-local, multicast, loopback.
func TestCalculate_SpecialRanges(t *testing.T) {
	cases := []struct {
		prefix      string
		linkLocal   bool
		multicast   bool
		loopback    bool
	}{
		{"169.254.0.0/16", true, false, false},
		{"169.254.42.0/24", true, false, false},
		{"169.253.0.0/16", false, false, false}, // boundary negative
		{"224.0.0.0/4", false, true, false},
		{"239.255.255.0/24", false, true, false}, // top of multicast
		{"240.0.0.0/4", false, false, false},     // class E, not multicast
		{"127.0.0.0/8", false, false, true},
		{"127.0.0.1/32", false, false, true},
		{"126.255.255.0/24", false, false, false}, // just below loopback
		{"128.0.0.0/8", false, false, false},      // just above loopback
	}
	for _, c := range cases {
		r := Calculate(netip.MustParsePrefix(c.prefix))
		if r.IsLinkLocal != c.linkLocal {
			t.Errorf("%s: IsLinkLocal got %v, want %v", c.prefix, r.IsLinkLocal, c.linkLocal)
		}
		if r.IsMulticast != c.multicast {
			t.Errorf("%s: IsMulticast got %v, want %v", c.prefix, r.IsMulticast, c.multicast)
		}
		if r.IsLoopback != c.loopback {
			t.Errorf("%s: IsLoopback got %v, want %v", c.prefix, r.IsLoopback, c.loopback)
		}
	}
}

// TestCalculate_BinaryRendering verifies the dotted-binary format used in
// the result panel. Each octet is exactly 8 bits, separator is a dot.
func TestCalculate_BinaryRendering(t *testing.T) {
	cases := []struct {
		prefix     string
		wantAddr   string
		wantMask   string
	}{
		{"0.0.0.0/0",
			"00000000.00000000.00000000.00000000",
			"00000000.00000000.00000000.00000000"},
		{"255.255.255.255/32",
			"11111111.11111111.11111111.11111111",
			"11111111.11111111.11111111.11111111"},
		{"10.20.30.40/24",
			"00001010.00010100.00011110.00101000",
			"11111111.11111111.11111111.00000000"},
		{"172.16.0.0/12",
			"10101100.00010000.00000000.00000000",
			"11111111.11110000.00000000.00000000"},
	}
	for _, c := range cases {
		r := Calculate(netip.MustParsePrefix(c.prefix))
		if r.BinaryAddr != c.wantAddr {
			t.Errorf("%s: BinaryAddr got %s, want %s", c.prefix, r.BinaryAddr, c.wantAddr)
		}
		if r.BinaryMask != c.wantMask {
			t.Errorf("%s: BinaryMask got %s, want %s", c.prefix, r.BinaryMask, c.wantMask)
		}
	}
}

// TestCalculate_ReverseDNS verifies the in-addr.arpa rendering — octets
// reversed, network address used (not the input).
func TestCalculate_ReverseDNS(t *testing.T) {
	cases := []struct {
		prefix string
		want   string
	}{
		{"192.168.1.0/24", "0.1.168.192.in-addr.arpa"},
		{"10.0.0.0/8", "0.0.0.10.in-addr.arpa"},
		{"1.2.3.4/32", "4.3.2.1.in-addr.arpa"},
		// Host bits set: reverseDNS is computed for the NETWORK, not the input.
		{"192.168.1.50/24", "0.1.168.192.in-addr.arpa"},
	}
	for _, c := range cases {
		r := Calculate(netip.MustParsePrefix(c.prefix))
		if r.ReverseDNS != c.want {
			t.Errorf("%s: ReverseDNS got %s, want %s", c.prefix, r.ReverseDNS, c.want)
		}
	}
}

// TestCalculate_HostCountAcrossPrefixes spot-checks host count at several
// prefix lengths. Catches off-by-ones in the bcastU-netU-1 formula.
func TestCalculate_HostCountAcrossPrefixes(t *testing.T) {
	cases := []struct {
		prefix string
		want   uint64
	}{
		{"10.0.0.0/8", (1 << 24) - 2}, // 16_777_214
		{"10.0.0.0/16", (1 << 16) - 2}, // 65_534
		{"10.0.0.0/24", 254},
		{"10.0.0.0/30", 2},
		{"10.0.0.0/31", 2}, // RFC 3021 special
		{"10.0.0.0/32", 1}, // single host
	}
	for _, c := range cases {
		r := Calculate(netip.MustParsePrefix(c.prefix))
		if r.HostCount != c.want {
			t.Errorf("%s: HostCount got %d, want %d", c.prefix, r.HostCount, c.want)
		}
	}
}

// TestCalculate_BinarySplit verifies that the network/host parts of the
// binary representation join back to the full string and split at the
// right bit, including non-octet-aligned prefixes.
func TestCalculate_BinarySplit(t *testing.T) {
	cases := []struct {
		prefix     string
		wantAddrNet, wantAddrHost string
	}{
		{"0.0.0.0/0", "", "00000000.00000000.00000000.00000000"},
		{"192.168.1.0/24",
			"11000000.10101000.00000001",
			".00000000"},
		{"192.168.1.0/26",
			"11000000.10101000.00000001.00",
			"000000"},
		{"10.0.0.1/32",
			"00001010.00000000.00000000.00000001",
			""},
	}
	for _, c := range cases {
		r := Calculate(netip.MustParsePrefix(c.prefix))
		if r.BinaryAddrNet != c.wantAddrNet {
			t.Errorf("%s: BinaryAddrNet got %q, want %q", c.prefix, r.BinaryAddrNet, c.wantAddrNet)
		}
		if r.BinaryAddrHost != c.wantAddrHost {
			t.Errorf("%s: BinaryAddrHost got %q, want %q", c.prefix, r.BinaryAddrHost, c.wantAddrHost)
		}
		if r.BinaryAddrNet+r.BinaryAddrHost != r.BinaryAddr {
			t.Errorf("%s: BinaryAddrNet+BinaryAddrHost != BinaryAddr (%q+%q != %q)",
				c.prefix, r.BinaryAddrNet, r.BinaryAddrHost, r.BinaryAddr)
		}
	}
}

// TestCalculate_NetmaskAndWildcard ensures the two are always inverses.
func TestCalculate_NetmaskAndWildcard(t *testing.T) {
	for bits := 0; bits <= 32; bits++ {
		prefix := netip.PrefixFrom(netip.MustParseAddr("10.0.0.0"), bits)
		r := Calculate(prefix)
		nm, wc := r.Netmask.As4(), r.Wildcard.As4()
		for i := 0; i < 4; i++ {
			if nm[i]^wc[i] != 0xFF {
				t.Errorf("/%d: Netmask %s and Wildcard %s are not inverses at octet %d (nm=%08b wc=%08b)",
					bits, r.Netmask, r.Wildcard, i, nm[i], wc[i])
				break
			}
		}
	}
}
