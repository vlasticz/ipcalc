package ipcalc

import (
	"fmt"
	"net/netip"
)

func isRFC1918(addr netip.Addr) bool {
	if !addr.Is4() {
		return false
	}
	b := addr.As4()
	switch {
	case b[0] == 10:
		return true
	case b[0] == 172 && b[1] >= 16 && b[1] <= 31:
		return true
	case b[0] == 192 && b[1] == 168:
		return true
	}
	return false
}

func isLinkLocal(addr netip.Addr) bool {
	b := addr.As4()
	return b[0] == 169 && b[1] == 254
}

func isMulticast(addr netip.Addr) bool {
	b := addr.As4()
	return b[0] >= 224 && b[0] <= 239
}

func isLoopback(addr netip.Addr) bool {
	b := addr.As4()
	return b[0] == 127
}

func reverseDNS(addr netip.Addr) string {
	b := addr.As4()
	return fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa", b[3], b[2], b[1], b[0])
}

// class returns the classful label (A..E) for an address. Empty when the
// prefix length makes classful labelling meaningless (/31, /32).
func class(addr netip.Addr, bits int) string {
	if bits >= 31 {
		return ""
	}
	b := addr.As4()
	switch {
	case b[0] < 128:
		return "A"
	case b[0] < 192:
		return "B"
	case b[0] < 224:
		return "C"
	case b[0] < 240:
		return "D"
	default:
		return "E"
	}
}
