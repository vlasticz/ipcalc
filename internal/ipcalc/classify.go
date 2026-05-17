package ipcalc

import (
	"fmt"
	"net/netip"
)

// IPv4 address ranges expressed as inclusive uint32 bounds. A prefix is in
// a category only if its entire [netU, bcastU] span lies within one of the
// listed ranges. This avoids false positives when a prefix is wider than
// the canonical block — e.g. 192.168.0.0/13 has its network address inside
// 192.168.0.0/16 but extends through 192.175.255.255 into public space, so
// it is NOT RFC1918.

var rfc1918Ranges = [][2]uint32{
	{0x0A000000, 0x0AFFFFFF}, // 10.0.0.0/8
	{0xAC100000, 0xAC1FFFFF}, // 172.16.0.0/12
	{0xC0A80000, 0xC0A8FFFF}, // 192.168.0.0/16
}

var linkLocalRanges = [][2]uint32{
	{0xA9FE0000, 0xA9FEFFFF}, // 169.254.0.0/16
}

var multicastRanges = [][2]uint32{
	{0xE0000000, 0xEFFFFFFF}, // 224.0.0.0/4
}

var loopbackRanges = [][2]uint32{
	{0x7F000000, 0x7FFFFFFF}, // 127.0.0.0/8
}

func inAnyRange(n, b uint32, rs [][2]uint32) bool {
	for _, r := range rs {
		if n >= r[0] && b <= r[1] {
			return true
		}
	}
	return false
}

func isRFC1918(n, b uint32) bool   { return inAnyRange(n, b, rfc1918Ranges) }
func isLinkLocal(n, b uint32) bool { return inAnyRange(n, b, linkLocalRanges) }
func isMulticast(n, b uint32) bool { return inAnyRange(n, b, multicastRanges) }
func isLoopback(n, b uint32) bool  { return inAnyRange(n, b, loopbackRanges) }

func reverseDNS(addr netip.Addr) string {
	b := addr.As4()
	return fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa", b[3], b[2], b[1], b[0])
}

// class returns A..E only when the entire prefix range lies within one
// classful block. Multi-class prefixes (e.g. /2 of 128.0.0.0 spanning B
// through E) return empty so the Reference row stays hidden rather than
// showing a label that is technically wrong.
func class(netU, bcastU uint32, bits int) string {
	if bits >= 31 {
		return ""
	}
	ranges := [...]struct {
		start, end uint32
		label      string
	}{
		{0x00000000, 0x7FFFFFFF, "A"},
		{0x80000000, 0xBFFFFFFF, "B"},
		{0xC0000000, 0xDFFFFFFF, "C"},
		{0xE0000000, 0xEFFFFFFF, "D"},
		{0xF0000000, 0xFFFFFFFF, "E"},
	}
	for _, r := range ranges {
		if netU >= r.start && bcastU <= r.end {
			return r.label
		}
	}
	return ""
}
