// Package external provides handlers for the external API.
package external

import (
	"net/http"
	"net/url"
	"strconv"

	"git.duti.dev/secure-package-registry/pkg/pkgdb"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// PackageHandler handles package-related routes.
type PackageHandler struct {
	db *pkgdb.Client
}

// NewPackageHandler creates a new PackageHandler.
func NewPackageHandler(db *pkgdb.Client) http.Handler {
	h := &PackageHandler{db: db}

	r := chi.NewRouter()
	r.Get("/", h.Search)                                       // GET /packages?q=...&ecosystem=...&page=...&page_size=...
	r.Get("/{ecosystem}/{identifier}/{version}", h.GetVersion) // GET /packages/{ecosystem}/{identifier}/{version}

	return r
}

// Search handles package search requests.
// q is optional - if empty, returns all packages (paginated).
func (h *PackageHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	// Parse optional ecosystem filter
	var ecosystem *pkgdb.Ecosystem
	if ecoStr := r.URL.Query().Get("ecosystem"); ecoStr != "" {
		eco := pkgdb.Ecosystem(ecoStr)
		ecosystem = &eco
	}

	// Parse pagination with defaults
	page := int32(1)
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = int32(p)
		}
	}

	pageSize := int32(20)
	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = int32(ps)
		}
	}

	result, err := h.db.SearchPackages(r.Context(), pkgdb.SearchParams{
		Query:     query,
		Ecosystem: ecosystem,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": "failed to search packages"})
		return
	}

	render.JSON(w, r, result)
}

// GetVersion handles requests for a specific package version.
func (h *PackageHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	ecosystem := pkgdb.Ecosystem(chi.URLParam(r, "ecosystem"))
	identifier, _ := url.PathUnescape(chi.URLParam(r, "identifier"))
	version := chi.URLParam(r, "version")

	pv, err := h.db.GetPackageVersion(r.Context(), ecosystem, identifier, version)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": "package version not found"})
		return
	}

	render.JSON(w, r, pv)
}
