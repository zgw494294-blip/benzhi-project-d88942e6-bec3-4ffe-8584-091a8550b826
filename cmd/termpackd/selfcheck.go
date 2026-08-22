package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"termpack/internal/application"
	"termpack/internal/domain"
)

type checkClient struct {
	baseURL string
	client  *http.Client
	serial  int
}

func (c *checkClient) command(ctx context.Context, method, path string, payload map[string]any) (application.PackView, error) {
	c.serial++
	payload["idempotencyKey"] = fmt.Sprintf("selfcheck-%02d", c.serial)
	body, err := json.Marshal(payload)
	if err != nil {
		return application.PackView{}, err
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return application.PackView{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return application.PackView{}, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return application.PackView{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return application.PackView{}, fmt.Errorf("%s %s 返回 %d: %s", method, path, response.StatusCode, raw)
	}
	var view application.PackView
	if err := json.Unmarshal(raw, &view); err != nil {
		return view, err
	}
	return view, nil
}

func (c *checkClient) health(ctx context.Context) error {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/healthz", nil)
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("健康检查返回 %d", response.StatusCode)
	}
	return nil
}

func expected(view application.PackView, status domain.TermPackStatus, revision int) error {
	if view.Pack.Status != status || view.Pack.CurrentRevision != revision {
		return fmt.Errorf("期望状态 %s/R%d，实际 %s/R%d", status, revision, view.Pack.Status, view.Pack.CurrentRevision)
	}
	return nil
}

func runSelfcheck(ctx context.Context, address string) error {
	client := &checkClient{baseURL: "http://" + address, client: &http.Client{Timeout: 3 * time.Second}}
	var err error
	for attempt := 0; attempt < 20; attempt++ {
		if err = client.health(ctx); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
	if err != nil {
		return fmt.Errorf("服务未就绪: %w", err)
	}
	view, err := client.command(ctx, http.MethodPost, "/api/v1/term-packs", map[string]any{"conferenceName": "全球清洁能源峰会", "sourceLanguage": "中文", "targetLanguage": "英语"})
	if err != nil {
		return err
	}
	packID := view.Pack.ID
	entries := []map[string]any{
		{"sourceTerm": "虚拟电厂", "preferredTranslation": "virtual power plant", "definition": "聚合分布式能源并参与电力系统调度的协调系统", "context": "圆桌讨论电网灵活性与需求响应", "evidence": "会议背景材料第 12 页"},
		{"sourceTerm": "绿证交易", "preferredTranslation": "green certificate trading", "definition": "可再生能源绿色电力证书的市场化流转", "context": "政策专场讨论跨境认可机制", "evidence": "主旨演讲讲稿第 4 节"},
	}
	for _, entry := range entries {
		entry["expectedVersion"] = view.Pack.Version
		view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/entries", entry)
		if err != nil {
			return err
		}
	}
	view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/submit", map[string]any{"expectedVersion": view.Pack.Version})
	if err != nil {
		return err
	}
	for _, entry := range view.Entries {
		view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/entries/"+entry.ID+"/review", map[string]any{"decision": "Accepted", "editorNote": "术语与会议材料一致", "expectedVersion": view.Pack.Version})
		if err != nil {
			return err
		}
	}
	if err := expected(view, domain.StatusReviewed, 1); err != nil {
		return err
	}
	view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/freeze", map[string]any{"expectedVersion": view.Pack.Version})
	if err != nil {
		return err
	}
	view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/rehearsal/start", map[string]any{"expectedVersion": view.Pack.Version})
	if err != nil {
		return err
	}
	view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/findings", map[string]any{"entryID": view.Entries[0].ID, "scenario": "同传高语速段落", "severity": "Major", "observation": "缩写首次出现时存在歧义", "expectedVersion": view.Pack.Version})
	if err != nil {
		return err
	}
	if err := expected(view, domain.StatusChangesRequested, 1); err != nil {
		return err
	}
	findingID := view.Findings[0].ID
	view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/findings/"+findingID+"/resolve", map[string]any{"resolution": "在口译准备页补充全称及缩写", "expectedVersion": view.Pack.Version})
	if err != nil {
		return err
	}
	view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/revisions", map[string]any{"expectedVersion": view.Pack.Version})
	if err != nil {
		return err
	}
	if err := expected(view, domain.StatusDraft, 2); err != nil {
		return err
	}
	corrected := view.Entries[0]
	view, err = client.command(ctx, http.MethodPatch, "/api/v1/term-packs/"+packID+"/entries/"+corrected.ID, map[string]any{"sourceTerm": corrected.SourceTerm, "preferredTranslation": corrected.PreferredTranslation, "definition": corrected.Definition, "context": corrected.Context + "；首次出现时读出全称", "evidence": corrected.Evidence, "expectedVersion": view.Pack.Version})
	if err != nil {
		return err
	}
	view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/submit", map[string]any{"expectedVersion": view.Pack.Version})
	if err != nil {
		return err
	}
	for _, entry := range view.Entries {
		view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/entries/"+entry.ID+"/review", map[string]any{"decision": "Accepted", "editorNote": "修订后复核通过", "expectedVersion": view.Pack.Version})
		if err != nil {
			return err
		}
	}
	view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/freeze", map[string]any{"expectedVersion": view.Pack.Version})
	if err != nil {
		return err
	}
	view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/rehearsal/start", map[string]any{"expectedVersion": view.Pack.Version})
	if err != nil {
		return err
	}
	view, err = client.command(ctx, http.MethodPost, "/api/v1/term-packs/"+packID+"/release", map[string]any{"approvedBy": "发布负责人 周岚", "expectedVersion": view.Pack.Version})
	if err != nil {
		return err
	}
	if err := expected(view, domain.StatusReleased, 2); err != nil {
		return err
	}
	if view.Certificate == nil || view.Certificate.EntryCount != 2 || view.Certificate.ContentDigest == "" {
		return fmt.Errorf("发布凭据不完整")
	}
	return nil
}
