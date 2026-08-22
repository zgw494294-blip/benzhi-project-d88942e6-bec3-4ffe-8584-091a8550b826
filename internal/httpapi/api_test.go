package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"termpack/internal/application"
	"termpack/internal/httpapi"
	"termpack/internal/store/sqlite"
	"termpack/internal/webui"
)

func TestAPIAndWorkbenchAreServedTogether(t *testing.T) {
	store, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	mux := http.NewServeMux()
	httpapi.New(application.NewService(store)).Register(mux)
	mux.Handle("GET /", webui.Handler())
	server := httptest.NewServer(httpapi.SecurityHeaders(mux))
	defer server.Close()
	response, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Security-Policy"), "default-src") {
		t.Fatal("工作台首页必须由带安全响应头的同一服务提供")
	}
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/api/v1/term-packs", strings.NewReader(`{"conferenceName":"会议","sourceLanguage":"中文","targetLanguage":"英语","idempotencyKey":"http-create","unexpected":true}`))
	request.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("未知 JSON 字段应返回 400，实际 %d", response.StatusCode)
	}
}
