// Package ipcalc derives every field of an IPv4 subnet calculation from a
// parsed netip.Prefix. The Result type is the single canonical output and
// covers everything kjokjo/ipcalc prints plus modern additions (RFC1918,
// link-local, multicast, reverse-DNS arpa).
//
// The package has no HTTP, DB, or template dependencies — calculation logic
// only. Errors live at the handler boundary; once you have a netip.Prefix,
// Calculate is total.
package ipcalc

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"strings"
)

// Result is the canonical output of a single-subnet calculation.
type Result struct {
	Input       netip.Prefix // as provided (preserves host bits)
	Normalized  netip.Prefix // host bits cleared
	HostBitsSet bool

	Network   netip.Addr
	Broadcast netip.Addr // zero Addr for /32; equals HostMax for /31
	HostMin   netip.Addr
	HostMax   netip.Addr
	HostCount uint64

	PrefixLen int
	Netmask   netip.Addr
	Wildcard  netip.Addr

	Class       string // "A".."E"; empty when not meaningfully classful (/31, /32)
	IsRFC1918   bool
	IsLinkLocal bool
	IsMulticast bool
	IsLoopback  bool
	IsPublic    bool // not in any of the recognised private/special ranges

	ReverseDNS  string // 0.1.168.192.in-addr.arpa for 192.168.1.0

	BinaryAddr string // "11000000.10101000.00000001.00000000"
	BinaryMask string

	// Network/host split for colour-coded binary rendering. BinaryAddrNet
	// covers the prefix bits, BinaryAddrHost covers the host bits. The
	// boundary "." goes with Host so Net ends on a real bit.
	BinaryAddrNet, BinaryAddrHost string
	BinaryMaskNet, BinaryMaskHost string
}

// Calculate derives every Result field from an IPv4 prefix. IPv6 is out of
// scope for v1 — the function returns a zero-but-input-populated Result if
// passed a non-IPv4 prefix; callers should validate at the handler boundary.
func Calculate(p netip.Prefix) Result {
	if !p.Addr().Is4() {
		return Result{Input: p, Normalized: p}
	}

	bits := p.Bits()
	addrU := addrToUint32(p.Addr())
	maskU := prefixToMask(bits)
	netU := addrU & maskU
	bcastU := netU | ^maskU

	network := uint32ToAddr(netU)
	binAddr := toBinaryDotted(addrU)
	binMask := toBinaryDotted(maskU)
	binAddrNet, binAddrHost := splitBinaryByPrefix(binAddr, bits)
	binMaskNet, binMaskHost := splitBinaryByPrefix(binMask, bits)

	r := Result{
		Input:          p,
		Normalized:     netip.PrefixFrom(network, bits),
		HostBitsSet:    addrU != netU,
		Network:        network,
		Netmask:        uint32ToAddr(maskU),
		Wildcard:       uint32ToAddr(^maskU),
		PrefixLen:      bits,
		BinaryAddr:     binAddr,
		BinaryMask:     binMask,
		BinaryAddrNet:  binAddrNet,
		BinaryAddrHost: binAddrHost,
		BinaryMaskNet:  binMaskNet,
		BinaryMaskHost: binMaskHost,
		IsRFC1918:      isRFC1918(netU, bcastU),
		IsLinkLocal:    isLinkLocal(netU, bcastU),
		IsMulticast:    isMulticast(netU, bcastU),
		IsLoopback:     isLoopback(netU, bcastU),
		IsPublic:       !isRFC1918(netU, bcastU) && !isLinkLocal(netU, bcastU) && !isMulticast(netU, bcastU) && !isLoopback(netU, bcastU),
		ReverseDNS:     reverseDNS(network),
		Class:          class(netU, bcastU, bits),
	}

	switch bits {
	case 32:
		r.HostCount = 1
		r.HostMin = network
		r.HostMax = network
	case 31:
		// RFC 3021: 2 usable hosts, no network/broadcast distinction.
		r.HostCount = 2
		r.HostMin = network
		r.HostMax = uint32ToAddr(netU + 1)
	default:
		r.HostCount = uint64(bcastU-netU) - 1
		r.HostMin = uint32ToAddr(netU + 1)
		r.HostMax = uint32ToAddr(bcastU - 1)
		r.Broadcast = uint32ToAddr(bcastU)
	}

	return r
}

func addrToUint32(a netip.Addr) uint32 {
	b := a.As4()
	return binary.BigEndian.Uint32(b[:])
}

func uint32ToAddr(u uint32) netip.Addr {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], u)
	return netip.AddrFrom4(b)
}

func prefixToMask(bits int) uint32 {
	if bits == 0 {
		return 0
	}
	return ^uint32(0) << (32 - bits)
}

func toBinaryDotted(u uint32) string {
	var parts [4]string
	for i := 0; i < 4; i++ {
		parts[i] = fmt.Sprintf("%08b", byte(u>>(24-8*i)))
	}
	return strings.Join(parts[:], ".")
}
