package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"

	"github.com/vlasticz/ipcalc/internal/storage"
)

// singleSaveInput is the JSON shape stored in saved.input_json for kind="single".
// Result is recomputed on view so saved rows stay forward-compatible.
type singleSaveInput struct {
	IP   string `json:"ip"`
	Mask string `json:"mask"`
}

// handleSavedList renders the list page with all current saves.
func (s *server) handleSavedList(w http.ResponseWriter, r *http.Request) {
	data := s.basePageData(r)
	rows, err := s.deps.Storage.ListSaved()
	if err != nil {
		s.deps.Logger.Error("list saved", "err", err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	items := make([]savedListItem, len(rows))
	for i, r := range rows {
		items[i] = savedListItem{Saved: r, Network: networkFromInput(r)}
	}
	data.SavedList = items
	s.renderPage(w, r, "page_saved.gohtml", data)
}

// networkFromInput returns the canonical "network/prefix" string for a
// saved row, computed from its stored input_json (kind="single"). Returns
// "—" if the row's kind is unknown or the JSON is malformed.
func networkFromInput(r *storage.Saved) string {
	if r.Kind != "single" {
		return "—"
	}
	var in singleSaveInput
	if err := json.Unmarshal([]byte(r.InputJSON), &in); err != nil {
		return "—"
	}
	addr, err := netip.ParseAddr(in.IP)
	if err != nil {
		return "—"
	}
	bits, err := parseMask(in.Mask)
	if err != nil {
		return "—"
	}
	return netip.PrefixFrom(addr, bits).Masked().String()
}

// handleSavedView renders a single saved calc, recomputing the Result from
// stored inputs.
func (s *server) handleSavedView(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	row, err := s.deps.Storage.GetSaved(slug)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		s.deps.Logger.Error("get saved", "slug", slug, "err", err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	data := s.basePageData(r)
	data.SavedItem = row
	data.Slug = row.Slug
	data.Label = row.Label

	if row.Kind == "single" {
		var in singleSaveInput
		if err := json.Unmarshal([]byte(row.InputJSON), &in); err == nil {
			data.IP = in.IP
			data.Mask = in.Mask
			if res, warn, cerr := computeFromForm(in.IP, in.Mask); cerr == nil {
				data.Result = res
				data.Warning = warn
			} else {
				data.Warning = cerr.Error()
			}
		}
	}

	s.renderPage(w, r, "page_saved_view.gohtml", data)
}

// handleSavedCreate handles POST /saved from the inline form on /calc.
// Returns a small HTML fragment (partial_save_success) that HTMX swaps
// into the save form's slot.
func (s *server) handleSavedCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	ip := strings.TrimSpace(r.PostFormValue("ip"))
	mask := strings.TrimSpace(r.PostFormValue("mask"))
	label := strings.TrimSpace(r.PostFormValue("label"))

	if ip == "" || mask == "" {
		http.Error(w, "ip and mask required", http.StatusBadRequest)
		return
	}
	// Validate before persisting — no point saving garbage.
	if _, err := netip.ParseAddr(ip); err != nil {
		http.Error(w, "invalid ip", http.StatusBadRequest)
		return
	}
	if _, err := parseMask(mask); err != nil {
		http.Error(w, "invalid mask", http.StatusBadRequest)
		return
	}

	body, _ := json.Marshal(singleSaveInput{IP: ip, Mask: mask})
	row, err := s.deps.Storage.CreateSaved(label, "single", string(body), nil)
	if err != nil {
		s.deps.Logger.Error("create saved", "err", err)
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}

	s.renderPartial(w, "partial_save_success", pageData{
		SaveResult: &savedResult{Slug: row.Slug, Label: row.Label},
	})
}

// handleSavedDelete handles DELETE /saved/{slug}. Returns 200 with empty
// body — the row's HTMX target swaps it out with empty content.
func (s *server) handleSavedDelete(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := s.deps.Storage.DeleteSaved(slug); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		s.deps.Logger.Error("delete saved", "slug", slug, "err", err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprint(w, "")
}
