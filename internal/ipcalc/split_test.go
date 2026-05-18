package ipcalc

import (
	"errors"
	"net/netip"
	"testing"
)

func TestSplitEqual_PowerOfTwo(t *testing.T) {
	// /24 split into 4 -> 4 × /26.
	got, err := SplitEqual(netip.MustParsePrefix("192.168.0.0/24"), 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{
		"192.168.0.0/26",
		"192.168.0.64/26",
		"192.168.0.128/26",
		"192.168.0.192/26",
	}
	if len(got) != len(want) {
		t.Fatalf("len got %d, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Normalized.String() != w {
			t.Errorf("[%d] got %s, want %s", i, got[i].Normalized, w)
		}
	}
}

func TestSplitEqual_NonPowerOfTwoRoundsUp(t *testing.T) {
	// /24 split into 5 -> smallest power of two >= 5 is 8 -> 8 × /27,
	// but we return only the first 5.
	got, err := SplitEqual(netip.MustParsePrefix("192.168.0.0/24"), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("len got %d, want 5", len(got))
	}
	want := []string{
		"192.168.0.0/27",
		"192.168.0.32/27",
		"192.168.0.64/27",
		"192.168.0.96/27",
		"192.168.0.128/27",
	}
	for i, w := range want {
		if got[i].Normalized.String() != w {
			t.Errorf("[%d] got %s, want %s", i, got[i].Normalized, w)
		}
	}
}

func TestSplitEqual_NEqualsOne(t *testing.T) {
	// Splitting into 1 returns the parent itself.
	got, err := SplitEqual(netip.MustParsePrefix("10.0.0.0/8"), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len got %d, want 1", len(got))
	}
	if got[0].Normalized.String() != "10.0.0.0/8" {
		t.Errorf("got %s, want 10.0.0.0/8", got[0].Normalized)
	}
}

func TestSplitEqual_DownToHosts(t *testing.T) {
	// /30 split into 4 -> 4 × /32.
	got, err := SplitEqual(netip.MustParsePrefix("10.0.0.0/30"), 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("len got %d, want 4", len(got))
	}
	if got[0].PrefixLen != 32 {
		t.Errorf("got /%d, want /32", got[0].PrefixLen)
	}
}

func TestSplitEqual_ErrTooMany(t *testing.T) {
	// /30 can hold 4 /32s; asking for 5 must error.
	_, err := SplitEqual(netip.MustParsePrefix("10.0.0.0/30"), 5)
	if !errors.Is(err, ErrSplitTooMany) {
		t.Errorf("expected ErrSplitTooMany, got %v", err)
	}
}

func TestSplitEqual_ErrInvalidN(t *testing.T) {
	for _, n := range []int{0, -1, -100} {
		if _, err := SplitEqual(netip.MustParsePrefix("10.0.0.0/24"), n); err == nil {
			t.Errorf("n=%d: expected error, got nil", n)
		}
	}
}

func TestSplitVLSM_TypicalAllocation(t *testing.T) {
	// /24 split for [50, 20, 10, 2] hosts.
	// /26 (50 ≤ 62), /27 (20 ≤ 30), /28 (10 ≤ 14), /31 (2 RFC3021).
	// Allocated largest-first; returned in caller order.
	got, err := SplitVLSM(netip.MustParsePrefix("192.168.0.0/24"), []int{50, 20, 10, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{
		"192.168.0.0/26",   // 50 hosts
		"192.168.0.64/27",  // 20 hosts
		"192.168.0.96/28",  // 10 hosts
		"192.168.0.112/31", // 2 hosts
	}
	for i, w := range want {
		if got[i].Normalized.String() != w {
			t.Errorf("[%d] got %s, want %s", i, got[i].Normalized, w)
		}
	}
}

func TestSplitVLSM_ReturnsInCallerOrder(t *testing.T) {
	// Even though VLSM allocates largest-first internally, results must
	// come back in the order the caller specified.
	got, err := SplitVLSM(netip.MustParsePrefix("192.168.0.0/24"), []int{20, 50, 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Internally allocates /26 first (for 50), then /27 (20), then /28 (10).
	// So in caller order: [0]=/27, [1]=/26, [2]=/28.
	want := []string{
		"192.168.0.64/27", // 20 hosts (allocated second)
		"192.168.0.0/26",  // 50 hosts (allocated first)
		"192.168.0.96/28", // 10 hosts (allocated third)
	}
	for i, w := range want {
		if got[i].Normalized.String() != w {
			t.Errorf("[%d] got %s, want %s", i, got[i].Normalized, w)
		}
	}
}

func TestSplitVLSM_ErrDoesNotFit(t *testing.T) {
	// /24 = 256 addresses. 8 × 50 hosts each need /26 (64 addresses) =
	// 512 total. Must not fit.
	_, err := SplitVLSM(
		netip.MustParsePrefix("192.168.0.0/24"),
		[]int{50, 50, 50, 50, 50, 50, 50, 50},
	)
	if !errors.Is(err, ErrVLSMDoesNotFit) {
		t.Errorf("expected ErrVLSMDoesNotFit, got %v", err)
	}
}

func TestSplitVLSM_SingleHost(t *testing.T) {
	// h=1 should pick /32.
	got, err := SplitVLSM(netip.MustParsePrefix("10.0.0.0/30"), []int{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].PrefixLen != 32 {
		t.Errorf("got /%d, want /32", got[0].PrefixLen)
	}
}

func TestSplitVLSM_TwoHostsUsesSlash31(t *testing.T) {
	// h=2 should pick /31 (RFC 3021), not /30.
	got, err := SplitVLSM(netip.MustParsePrefix("10.0.0.0/29"), []int{2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].PrefixLen != 31 {
		t.Errorf("got /%d, want /31", got[0].PrefixLen)
	}
}

func TestSplitVLSM_ErrInvalidInput(t *testing.T) {
	parent := netip.MustParsePrefix("10.0.0.0/24")
	if _, err := SplitVLSM(parent, []int{}); err == nil {
		t.Error("empty hosts: expected error, got nil")
	}
	if _, err := SplitVLSM(parent, []int{0}); err == nil {
		t.Error("h=0: expected error, got nil")
	}
	if _, err := SplitVLSM(parent, []int{-5}); err == nil {
		t.Error("h=-5: expected error, got nil")
	}
}

func TestPrefixBitsForHosts(t *testing.T) {
	cases := []struct {
		h    int
		want int
	}{
		{1, 32},
		{2, 31},
		{3, 29},  // 6 usable
		{6, 29},
		{7, 28},  // 14 usable
		{14, 28},
		{15, 27}, // 30 usable
		{30, 27},
		{31, 26}, // 62 usable
		{62, 26},
		{63, 25}, // 126 usable
		{126, 25},
		{127, 24}, // 254 usable
		{254, 24},
	}
	for _, c := range cases {
		got := prefixBitsForHosts(c.h)
		if got != c.want {
			t.Errorf("prefixBitsForHosts(%d) got /%d, want /%d", c.h, got, c.want)
		}
	}
}
