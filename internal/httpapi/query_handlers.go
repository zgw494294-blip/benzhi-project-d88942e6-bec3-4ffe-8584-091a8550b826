package httpapi

import (
	"fmt"
	"net/http"
	"strconv"

	"termpack/internal/application"
	"termpack/internal/domain"
)

func (a *API) HealthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) ListTermPacksHandler(w http.ResponseWriter, r *http.Request) {
	packs, err := a.service.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"termPacks": packs})
}

func (a *API) GetTermPackHandler(w http.ResponseWriter, r *http.Request) {
	view, err := a.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (a *API) CertificateHandler(w http.ResponseWriter, r *http.Request) {
	view, err := a.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if view.Certificate == nil {
		writeError(w, domainNotFound("发布凭据尚未生成"))
		return
	}
	writeJSON(w, http.StatusOK, view.Certificate)
}

func (a *API) VerifyCertificateHandler(w http.ResponseWriter, r *http.Request) {
	result, err := a.service.VerifyCertificate(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) PreflightHandler(w http.ResponseWriter, r *http.Request) {
	revision, err := optionalInt(r.URL.Query().Get("revision"))
	if err != nil {
		writeError(w, domain.NewRuleError("invalid_revision", "revision 必须是正整数"))
		return
	}
	report, err := a.service.Preflight(r.Context(), r.PathValue("id"), revision)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (a *API) FindingsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	revision, err := optionalInt(query.Get("frozenRevision"))
	if err != nil {
		writeError(w, domain.NewRuleError("invalid_revision", "frozenRevision 必须是正整数"))
		return
	}
	severity := domain.FindingSeverity(query.Get("severity"))
	if severity != "" && severity != domain.SeverityMinor && severity != domain.SeverityMajor && severity != domain.SeverityCritical {
		writeError(w, domain.NewRuleError("invalid_filter", "severity 筛选值无效"))
		return
	}
	status := domain.FindingStatus(query.Get("status"))
	if status != "" && status != domain.FindingOpen && status != domain.FindingResolved {
		writeError(w, domain.NewRuleError("invalid_filter", "status 筛选值无效"))
		return
	}
	report, err := a.service.FindingsReport(r.Context(), application.FindingFilter{PackID: r.PathValue("id"), FrozenRevision: revision, Severity: severity, Status: status})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (a *API) RevisionDiffHandler(w http.ResponseWriter, r *http.Request) {
	current, err := strconv.Atoi(r.PathValue("revision"))
	if err != nil {
		writeError(w, domain.ErrNotFound)
		return
	}
	previous, err := optionalInt(r.URL.Query().Get("previousRevision"))
	if err != nil {
		writeError(w, domain.NewRuleError("invalid_revision", "previousRevision 必须是正整数"))
		return
	}
	diff, err := a.service.RevisionDiff(r.Context(), r.PathValue("id"), current, previous)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, diff)
}

func optionalInt(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		if err == nil {
			err = fmt.Errorf("revision must be positive")
		}
		return 0, err
	}
	return value, nil
}

func domainNotFound(message string) error {
	return domain.NewRuleError("not_found", message)
}
