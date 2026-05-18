package ipcalc

import (
	"errors"
	"fmt"
	"net/netip"
	"sort"
)

var (
	ErrSplitTooMany   = errors.New("more subnets requested than the parent can hold")
	ErrVLSMDoesNotFit = errors.New("requested host allocations do not fit in the parent network")
	ErrInvalidParent  = errors.New("parent prefix is not a valid IPv4 network")
)

// MaxSubnets is a soft cap on split results to keep response/render sizes
// sane. ~4 KiB per Result × 4096 = ~16 MiB worst case; well over what any
// homelab tool needs.
const MaxSubnets = 4096

// SplitEqual divides parent into n equally-sized subnets. The chosen child
// prefix length is the smallest that yields >= n subnets — i.e.
// parentBits + ceilLog2(n). Returns exactly n Results (not all possible
// subnets at that prefix length).
func SplitEqual(parent netip.Prefix, n int) ([]Result, error) {
	if !parent.Addr().Is4() {
		return nil, ErrInvalidParent
	}
	if n <= 0 {
		return nil, errors.New("n must be positive")
	}
	if n > MaxSubnets {
		return nil, fmt.Errorf("n=%d exceeds MaxSubnets=%d", n, MaxSubnets)
	}

	parentBits := parent.Bits()
	childBits := parentBits + ceilLog2(n)
	if childBits > 32 {
		return nil, ErrSplitTooMany
	}

	parentNetU := addrToUint32(parent.Addr()) & prefixToMask(parentBits)
	childSize := uint64(1) << (32 - childBits)

	out := make([]Result, n)
	for i := 0; i < n; i++ {
		childNetU := parentNetU + uint32(uint64(i)*childSize)
		out[i] = Calculate(netip.PrefixFrom(uint32ToAddr(childNetU), childBits))
	}
	return out, nil
}

// SplitVLSM packs the requested host-count allocations into parent using
// variable-length subnet masking. Internally allocates largest-first to
// avoid fragmentation, then returns Results in caller-supplied order so
// labels (e.g. "VLAN A=50 hosts") line up with results[i].
func SplitVLSM(parent netip.Prefix, hosts []int) ([]Result, error) {
	if !parent.Addr().Is4() {
		return nil, ErrInvalidParent
	}
	if len(hosts) == 0 {
		return nil, errors.New("no host counts provided")
	}
	if len(hosts) > MaxSubnets {
		return nil, fmt.Errorf("%d allocations exceeds MaxSubnets=%d", len(hosts), MaxSubnets)
	}

	parentBits := parent.Bits()
	parentNetU := addrToUint32(parent.Addr()) & prefixToMask(parentBits)
	parentBcastU := parentNetU | ^prefixToMask(parentBits)

	type req struct {
		idx   int
		hosts int
		bits  int
		size  uint64
	}
	reqs := make([]req, len(hosts))
	for i, h := range hosts {
		if h <= 0 {
			return nil, fmt.Errorf("host count must be positive (got %d at index %d)", h, i)
		}
		bits := prefixBitsForHosts(h)
		if bits < parentBits {
			return nil, fmt.Errorf("host count %d needs /%d which is wider than parent /%d", h, bits, parentBits)
		}
		reqs[i] = req{idx: i, hosts: h, bits: bits, size: uint64(1) << (32 - bits)}
	}

	// Sort largest-first (smallest prefix length = largest subnet size).
	// Stable so equal-size requests keep their original order.
	sorted := append([]req(nil), reqs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].bits < sorted[j].bits
	})

	allocations := make([]netip.Prefix, len(hosts))
	cursor := uint64(parentNetU)
	parentEnd := uint64(parentBcastU)
	for _, r := range sorted {
		// Align cursor up to the subnet's size boundary.
		if rem := cursor % r.size; rem != 0 {
			cursor += r.size - rem
		}
		if cursor+r.size-1 > parentEnd {
			return nil, ErrVLSMDoesNotFit
		}
		allocations[r.idx] = netip.PrefixFrom(uint32ToAddr(uint32(cursor)), r.bits)
		cursor += r.size
	}

	out := make([]Result, len(hosts))
	for i, p := range allocations {
		out[i] = Calculate(p)
	}
	return out, nil
}

// prefixBitsForHosts returns the smallest prefix length that fits h usable
// host addresses, accounting for network + broadcast overhead at /<=30.
// Special cases: h=1 -> /32 (single host), h=2 -> /31 (RFC 3021, no
// network/broadcast distinction).
func prefixBitsForHosts(h int) int {
	if h <= 1 {
		return 32
	}
	if h <= 2 {
		return 31
	}
	// Smallest k such that 2^k - 2 >= h, i.e. 2^k >= h + 2.
	return 32 - ceilLog2(h+2)
}

// ceilLog2 returns ceil(log2(n)) for n >= 1; 0 for n <= 1.
func ceilLog2(n int) int {
	if n <= 1 {
		return 0
	}
	bits := 0
	n--
	for n > 0 {
		n >>= 1
		bits++
	}
	return bits
}
