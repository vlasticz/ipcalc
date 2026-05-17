package server

import (
	"fmt"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"github.com/vlasticz/ipcalc/internal/ipcalc"
)

// handleCalcPage renders the full calculator page. Optional query params
// `ip` and `mask` pre-fill the form and pre-compute a result so URLs are
// shareable without saving.
func (s *server) handleCalcPage(w http.ResponseWriter, r *http.Request) {
	data := s.basePageData(r)
	ip := strings.TrimSpace(r.URL.Query().Get("ip"))
	mask := strings.TrimSpace(r.URL.Query().Get("mask"))
	data.IP, data.Mask = ip, mask

	if ip != "" && mask != "" {
		if res, warn, err := computeFromForm(ip, mask); err == nil {
			data.Result = res
			data.Warning = warn
		} else {
			data.Warning = err.Error()
		}
	}
	s.renderPage(w, r, "page_calc.gohtml", data)
}

// handleCalcFragment serves the HTMX fragment for the result panel.
func (s *server) handleCalcFragment(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	ip := strings.TrimSpace(r.PostFormValue("ip"))
	mask := strings.TrimSpace(r.PostFormValue("mask"))

	// Empty fields render an empty fragment — HTMX swap just clears the panel.
	if ip == "" || mask == "" {
		s.renderPartial(w, "partial_result", pageData{})
		return
	}

	res, warn, err := computeFromForm(ip, mask)
	data := pageData{}
	if err != nil {
		data.Warning = err.Error()
	} else {
		data.Result = res
		data.Warning = warn
	}
	s.renderPartial(w, "partial_result", data)
}

// computeFromForm parses an IP + mask pair (where mask may be CIDR-form like
// "24" or dotted-quad like "255.255.255.0") and returns a Result. Forgiving:
// if host bits are set, normalises and returns a warning string.
func computeFromForm(ipStr, maskStr string) (*ipcalc.Result, string, error) {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return nil, "", fmt.Errorf("invalid IP address: %s", ipStr)
	}
	if !addr.Is4() {
		return nil, "", fmt.Errorf("IPv6 is not supported in v1")
	}

	bits, err := parseMask(maskStr)
	if err != nil {
		return nil, "", err
	}

	prefix := netip.PrefixFrom(addr, bits)
	result := ipcalc.Calculate(prefix)

	var warning string
	if result.HostBitsSet {
		warning = fmt.Sprintf("Host bits set; computing for network %s/%d.", result.Network, result.PrefixLen)
	}
	return &result, warning, nil
}

// parseMask accepts CIDR length ("0".."32") or dotted-quad netmask
// ("255.255.255.0") and returns the prefix length.
func parseMask(s string) (int, error) {
	if n, err := strconv.Atoi(s); err == nil {
		if n < 0 || n > 32 {
			return 0, fmt.Errorf("mask out of range: %d", n)
		}
		return n, nil
	}
	addr, err := netip.ParseAddr(s)
	if err != nil || !addr.Is4() {
		return 0, fmt.Errorf("invalid mask: %s", s)
	}
	b := addr.As4()
	bits := 0
	prev := byte(0xFF)
	for _, oct := range b {
		// Reject non-contiguous masks like 255.0.255.0.
		if oct > prev {
			return 0, fmt.Errorf("non-contiguous netmask: %s", s)
		}
		bits += countLeadingOnes(oct)
		prev = oct
	}
	return bits, nil
}

func countLeadingOnes(b byte) int {
	switch b {
	case 0xFF:
		return 8
	case 0xFE:
		return 7
	case 0xFC:
		return 6
	case 0xF8:
		return 5
	case 0xF0:
		return 4
	case 0xE0:
		return 3
	case 0xC0:
		return 2
	case 0x80:
		return 1
	case 0x00:
		return 0
	default:
		return -1
	}
}
