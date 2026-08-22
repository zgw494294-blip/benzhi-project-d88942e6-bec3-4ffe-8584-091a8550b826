package httpapi

import (
	"net/http"

	"termpack/internal/application"
)

type API struct {
	service             *application.Service
	batchEntriesRequest batchEntriesRequest
	batchReviewRequest  batchReviewRequest
	batchResolveRequest batchResolveRequest
}

func New(service *application.Service) *API { return &API{service: service} }

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", a.HealthHandler)
	mux.HandleFunc("GET /api/v1/term-packs", a.ListTermPacksHandler)
	mux.HandleFunc("POST /api/v1/term-packs", a.CreateTermPackHandler)
	mux.HandleFunc("GET /api/v1/term-packs/{id}", a.GetTermPackHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/entries", a.AddEntryHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/entries/batch", a.BatchAddEntriesHandler)
	mux.HandleFunc("PATCH /api/v1/term-packs/{id}/entries/{entryID}", a.UpdateEntryHandler)
	mux.HandleFunc("PATCH /api/v1/term-packs/{id}/metadata", a.UpdateMetadataHandler)
	mux.HandleFunc("PATCH /api/v1/term-packs/{id}", a.UpdateMetadataHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/submit", a.SubmitHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/entries/{entryID}/review", a.ReviewEntryHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/entries/review-batch", a.BatchReviewHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/entries/batch-review", a.BatchReviewHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/freeze", a.FreezeHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/rehearsal/start", a.StartRehearsalHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/findings", a.AddFindingHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/findings/{findingID}/resolve", a.ResolveFindingHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/findings/resolve-batch", a.BatchResolveFindingHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/findings/batch-resolve", a.BatchResolveFindingHandler)
	mux.HandleFunc("GET /api/v1/term-packs/{id}/findings", a.FindingsHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/revisions", a.BeginRevisionHandler)
	mux.HandleFunc("POST /api/v1/term-packs/{id}/release", a.ReleaseHandler)
	mux.HandleFunc("GET /api/v1/term-packs/{id}/certificate", a.CertificateHandler)
	mux.HandleFunc("GET /api/v1/term-packs/{id}/certificate/verify", a.VerifyCertificateHandler)
	mux.HandleFunc("GET /api/v1/term-packs/{id}/preflight", a.PreflightHandler)
	mux.HandleFunc("GET /api/v1/term-packs/{id}/precheck", a.PreflightHandler)
	mux.HandleFunc("GET /api/v1/term-packs/{id}/revisions/{revision}/diff", a.RevisionDiffHandler)
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}
