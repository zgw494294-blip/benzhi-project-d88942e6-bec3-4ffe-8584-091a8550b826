package httpapi

import (
	"net/http"

	"termpack/internal/application"
)

func (a *API) ReviewEntryHandler(w http.ResponseWriter, r *http.Request) {
	var request reviewEntryRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	view, err := a.service.ReviewEntry(r.Context(), application.ReviewEntry{PackID: r.PathValue("id"), EntryID: r.PathValue("entryID"), Decision: request.Decision, Translation: request.Translation, EditorNote: request.EditorNote, ExpectedVersion: request.ExpectedVersion, IdempotencyKey: request.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (a *API) BatchReviewHandler(w http.ResponseWriter, r *http.Request) {
	var request batchReviewRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	view, err := a.service.ReviewEntries(r.Context(), application.BatchReview{PackID: r.PathValue("id"), Items: request.Items, ExpectedVersion: request.ExpectedVersion, IdempotencyKey: request.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (a *API) AddFindingHandler(w http.ResponseWriter, r *http.Request) {
	var request addFindingRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	view, err := a.service.AddFinding(r.Context(), application.AddFinding{PackID: r.PathValue("id"), EntryID: request.EntryID, Scenario: request.Scenario, Severity: request.Severity, Observation: request.Observation, ExpectedVersion: request.ExpectedVersion, IdempotencyKey: request.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (a *API) ResolveFindingHandler(w http.ResponseWriter, r *http.Request) {
	var request resolveFindingRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	view, err := a.service.ResolveFinding(r.Context(), application.ResolveFinding{PackID: r.PathValue("id"), FindingID: r.PathValue("findingID"), Resolution: request.Resolution, ExpectedVersion: request.ExpectedVersion, IdempotencyKey: request.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (a *API) BatchResolveFindingHandler(w http.ResponseWriter, r *http.Request) {
	var request batchResolveRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	view, err := a.service.CloseFindings(r.Context(), application.CloseFindings{PackID: r.PathValue("id"), Items: request.Items, ExpectedVersion: request.ExpectedVersion, IdempotencyKey: request.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (a *API) ReleaseHandler(w http.ResponseWriter, r *http.Request) {
	var request releaseRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	view, err := a.service.Release(r.Context(), application.Release{PackID: r.PathValue("id"), ApprovedBy: request.ApprovedBy, ExpectedVersion: request.ExpectedVersion, IdempotencyKey: request.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}
