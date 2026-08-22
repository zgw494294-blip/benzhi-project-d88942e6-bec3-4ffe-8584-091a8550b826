package httpapi

import (
	"net/http"

	"termpack/internal/application"
)

func (a *API) CreateTermPackHandler(w http.ResponseWriter, r *http.Request) {
	var request createPackRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	view, err := a.service.Create(r.Context(), application.CreatePack{ConferenceName: request.ConferenceName, SourceLanguage: request.SourceLanguage, TargetLanguage: request.TargetLanguage, IdempotencyKey: request.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (a *API) AddEntryHandler(w http.ResponseWriter, r *http.Request) {
	var request addEntryRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	view, err := a.service.AddEntry(r.Context(), application.AddEntry{PackID: r.PathValue("id"), SourceTerm: request.SourceTerm, PreferredTranslation: request.PreferredTranslation, Definition: request.Definition, Context: request.Context, Evidence: request.Evidence, ExpectedVersion: request.ExpectedVersion, IdempotencyKey: request.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (a *API) BatchAddEntriesHandler(w http.ResponseWriter, r *http.Request) {
	var request batchEntriesRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	view, err := a.service.AddEntries(r.Context(), application.BatchAddEntries{PackID: r.PathValue("id"), Entries: request.Entries, ExpectedVersion: request.ExpectedVersion, IdempotencyKey: request.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (a *API) UpdateMetadataHandler(w http.ResponseWriter, r *http.Request) {
	var request metadataRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	view, err := a.service.UpdateMetadata(r.Context(), application.UpdateMetadata{PackID: r.PathValue("id"), ConferenceName: request.ConferenceName, SourceLanguage: request.SourceLanguage, TargetLanguage: request.TargetLanguage, ExpectedVersion: request.ExpectedVersion, IdempotencyKey: request.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (a *API) UpdateEntryHandler(w http.ResponseWriter, r *http.Request) {
	var request updateEntryRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	view, err := a.service.UpdateEntry(r.Context(), application.UpdateEntry{PackID: r.PathValue("id"), EntryID: r.PathValue("entryID"), SourceTerm: request.SourceTerm, PreferredTranslation: request.PreferredTranslation, Definition: request.Definition, Context: request.Context, Evidence: request.Evidence, ExpectedVersion: request.ExpectedVersion, IdempotencyKey: request.IdempotencyKey})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func versioned(r *http.Request, request versionedRequest) application.VersionedCommand {
	return application.VersionedCommand{PackID: r.PathValue("id"), ExpectedVersion: request.ExpectedVersion, IdempotencyKey: request.IdempotencyKey}
}

func (a *API) runVersioned(w http.ResponseWriter, r *http.Request, run func(application.VersionedCommand) (any, error)) {
	var request versionedRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	result, err := run(versioned(r, request))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) SubmitHandler(w http.ResponseWriter, r *http.Request) {
	a.runVersioned(w, r, func(cmd application.VersionedCommand) (any, error) { return a.service.Submit(r.Context(), cmd) })
}

func (a *API) FreezeHandler(w http.ResponseWriter, r *http.Request) {
	a.runVersioned(w, r, func(cmd application.VersionedCommand) (any, error) { return a.service.Freeze(r.Context(), cmd) })
}

func (a *API) StartRehearsalHandler(w http.ResponseWriter, r *http.Request) {
	a.runVersioned(w, r, func(cmd application.VersionedCommand) (any, error) { return a.service.StartRehearsal(r.Context(), cmd) })
}

func (a *API) BeginRevisionHandler(w http.ResponseWriter, r *http.Request) {
	a.runVersioned(w, r, func(cmd application.VersionedCommand) (any, error) { return a.service.BeginRevision(r.Context(), cmd) })
}
