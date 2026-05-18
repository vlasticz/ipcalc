package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"github.com/vlasticz/ipcalc/internal/ipcalc"
)

// handleSplitPage renders the full /split page with both Equal and VLSM
// tab panels. The active panel defaults to Equal — switched client-side.
//
// Accepts URL query parameters for cross-page prefill from /calc:
//   - ?parent=192.168.0.0/24  (direct CIDR)
//   - ?ip=192.168.0.0&mask=24 or ?ip=...&mask=255.255.255.0
// Both forms (Equal and VLSM) share the same parent value via SplitParent.
func (s *server) handleSplitPage(w http.ResponseWriter, r *http.Request) {
	data := s.basePageData(r)
	data.SplitMode = "equal"

	q := r.URL.Query()
	if parent := strings.TrimSpace(q.Get("parent")); parent != "" {
		data.SplitParent = parent
	} else if ip := strings.TrimSpace(q.Get("ip")); ip != "" {
		if maskStr := strings.TrimSpace(q.Get("mask")); maskStr != "" {
			// parseMask (handlers_calc.go) accepts either CIDR length or
			// dotted netmask; we normalise to CIDR length here so the
			// prefilled parent value is always valid Prefix syntax.
			if bits, err := parseMask(maskStr); err == nil {
				data.SplitParent = ip + "/" + strconv.Itoa(bits)
			} else {
				// Best-effort: hand back what the user gave so they see why it failed.
				data.SplitParent = ip + "/" + maskStr
			}
		}
	}

	s.renderPage(w, r, "page_split.gohtml", data)
}

// handleSplitEqual answers POST /split/equal with the result fragment.
// hx-target on the form is #result-equal, hx-swap=outerHTML.
func (s *server) handleSplitEqual(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	parent := strings.TrimSpace(r.PostFormValue("parent"))
	nStr := strings.TrimSpace(r.PostFormValue("n"))

	data := pageData{SplitMode: "equal", SplitParent: parent, SplitN: nStr}

	if parent == "" || nStr == "" {
		s.renderPartial(w, "partial_split_result", data)
		return
	}

	p, err := parseIPv4Prefix(parent)
	if err != nil {
		data.SplitError = err.Error()
		s.renderPartial(w, "partial_split_result", data)
		return
	}

	n, err := strconv.Atoi(nStr)
	if err != nil {
		data.SplitError = fmt.Sprintf("invalid subnet count: %q", nStr)
		s.renderPartial(w, "partial_split_result", data)
		return
	}

	results, err := ipcalc.SplitEqual(p, n)
	if err != nil {
		data.SplitError = err.Error()
		s.renderPartial(w, "partial_split_result", data)
		return
	}
	data.SplitResults = results
	s.renderPartial(w, "partial_split_result", data)
}

// handleSplitVLSM answers POST /split/vlsm. Host counts are comma-separated.
func (s *server) handleSplitVLSM(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	parent := strings.TrimSpace(r.PostFormValue("parent"))
	hostsStr := strings.TrimSpace(r.PostFormValue("hosts"))

	data := pageData{SplitMode: "vlsm", SplitParent: parent, SplitHosts: hostsStr}

	if parent == "" || hostsStr == "" {
		s.renderPartial(w, "partial_split_result", data)
		return
	}

	p, err := parseIPv4Prefix(parent)
	if err != nil {
		data.SplitError = err.Error()
		s.renderPartial(w, "partial_split_result", data)
		return
	}

	hosts, err := parseHostsList(hostsStr)
	if err != nil {
		data.SplitError = err.Error()
		s.renderPartial(w, "partial_split_result", data)
		return
	}

	results, err := ipcalc.SplitVLSM(p, hosts)
	if err != nil {
		data.SplitError = err.Error()
		s.renderPartial(w, "partial_split_result", data)
		return
	}
	data.SplitResults = results
	data.SplitRequested = hosts
	s.renderPartial(w, "partial_split_result", data)
}

// parseIPv4Prefix accepts CIDR notation (e.g. "192.168.0.0/24") and
// rejects IPv6. Host bits are silently masked off — the prefix is treated
// as the network it implies.
func parseIPv4Prefix(s string) (netip.Prefix, error) {
	p, err := netip.ParsePrefix(s)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("invalid parent prefix: %q", s)
	}
	if !p.Addr().Is4() {
		return netip.Prefix{}, errors.New("IPv6 is not supported in v1")
	}
	return p.Masked(), nil
}

func parseHostsList(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	hosts := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		h, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid host count: %q", p)
		}
		hosts = append(hosts, h)
	}
	if len(hosts) == 0 {
		return nil, errors.New("no host counts provided")
	}
	return hosts, nil
}
