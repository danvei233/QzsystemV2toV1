package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"xiaoheiproxy/internal/domain"
	cacheinfra "xiaoheiproxy/internal/infrastructure/cache"
	"xiaoheiproxy/internal/infrastructure/upstream"
)

type noopLogRepo struct{}

func (noopLogRepo) Create(context.Context, *domain.RequestLog) error { return nil }
func (noopLogRepo) List(context.Context, domain.RequestLogFilter) ([]domain.RequestLog, int64, error) {
	return nil, 0, nil
}
func (noopLogRepo) Get(context.Context, uint) (*domain.RequestLog, error) { return nil, nil }
func (noopLogRepo) EndpointErrorStats(context.Context, domain.RequestLogFilter, int) ([]domain.EndpointErrorStat, error) {
	return nil, nil
}

type memoryMetadataRepo struct {
	mu    sync.Mutex
	items map[uint]domain.HostV2Metadata
}

func newMemoryMetadataRepo() *memoryMetadataRepo {
	return &memoryMetadataRepo{items: map[uint]domain.HostV2Metadata{}}
}

func (r *memoryMetadataRepo) Upsert(_ context.Context, metadata *domain.HostV2Metadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[metadata.HostID] = *metadata
	return nil
}

func (r *memoryMetadataRepo) Get(_ context.Context, hostID uint) (*domain.HostV2Metadata, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[hostID]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (r *memoryMetadataRepo) Delete(_ context.Context, hostID uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, hostID)
	return nil
}

func TestServiceOpenHostFetchesHostInfoAndAppliesMetadata(t *testing.T) {
	var createHostCount, hostInfoCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/create_host"):
			createHostCount++
			writeJSON(w, map[string]any{
				"code": 1,
				"msg":  "succ",
				"data": map[string]any{
					"id":        911418,
					"host_name": "ser226939841627",
				},
			})
		case strings.Contains(r.URL.Path, "/hostinfo"):
			hostInfoCount++
			writeJSON(w, map[string]any{
				"code": 1,
				"msg":  "succ",
				"data": map[string]any{
					"id":              911418,
					"area_id":         3,
					"area_name":       "山西",
					"line_name":       "湖北电信100G防御一区",
					"node_name":       "高频服务器节点1",
					"line_id":         7,
					"node_id":         8,
					"host_name":       "ser226939841627",
					"ip":              "192.168.9.59",
					"local_ip":        "192.168.30.68",
					"buy_time":        "2026-05-25 17:09:58",
					"end_time":        "2026-05-25 00:49:50",
					"cpu":             4,
					"memory":          8192,
					"hard_disks":      120,
					"bandwidth":       10,
					"os_name":         "Windows Server 2022",
					"os_password":     "Qz123456",
					"panel_password":  "Qz123456",
					"cpu_limit":       100,
					"state":           1,
					"close_network":   "1",
					"traffic":         0,
					"reinstall_num":   0,
					"os_disk_maxiops": "0",
					"data_disk_iops":  0,
					"sync_time":       0,
					"now_iso":         "",
					"snapshot_num":    0,
					"backup_num":      1,
					"domain_num":      0,
					"bios":            "",
					"metal":           0,
					"is_nat":          1,
					"port_num":        10,
					"mac":             "",
					"mac1":            "",
					"remote_ip":       "",
				},
			})
		default:
			t.Fatalf("unexpected upstream path: %s", r.URL.String())
		}
	}))
	defer server.Close()

	metaRepo := newMemoryMetadataRepo()
	svc := NewService(
		upstream.NewClient(server.URL, "apikey", 5*time.Second),
		noopLogRepo{},
		metaRepo,
		cacheinfra.NewMemoryCache(),
		30*time.Second,
		1<<20,
		nil,
	)

	body := []byte(`{"line_id":"7","nodes_id":"8","os_name":"Windows Server 2022","host_name":"ser226939841627","cpu":"4","cpu_limit":"100","memory":"8192","sys_disk_size":"50","data_disk_size":"100","sys_disk_iops":"0","data_disk_iops":"0","net_out":"20","net_in":"100","snapshot":"0","backups":"1","max_reinstall_num":"5","expire_time":"2026-05-25 00:49:50","flow_limit":"40","is_nat":1,"port_num":10,"domain_num":0}`)
	resp, err := svc.Handle(context.Background(), RequestContext{
		Method: http.MethodPost,
		Path:   "/api/v1/openHost",
		Body:   body,
	})
	if err != nil {
		t.Fatalf("Handle openHost: %v", err)
	}

	payload := decodeJSONMap(t, resp.Body)
	data := payload["data"].(map[string]any)
	if intFromAny(payload["code"]) != 200 || stringFromAny(payload["msg"]) != "success" {
		t.Fatalf("unexpected envelope: %#v", payload)
	}
	if intFromAny(data["memory"]) != 8 {
		t.Fatalf("expected memory 8, got %#v", data["memory"])
	}
	if intFromAny(data["os_size"]) != 50 || intFromAny(data["hard_disks"]) != 100 {
		t.Fatalf("expected os_size 50 and hard_disks 100, got %#v", data)
	}
	if intFromAny(data["bandwidth_out"]) != 20 || intFromAny(data["bandwidth_in"]) != 100 {
		t.Fatalf("expected bandwidth override, got %#v", data)
	}
	if stringFromAny(data["area_id"]) != "湖北电信100G防御一区" || stringFromAny(data["area_name"]) != "高频服务器节点1" {
		t.Fatalf("expected line/node naming, got area_id=%#v area_name=%#v", data["area_id"], data["area_name"])
	}
	if stringFromAny(data["os_password"]) != "Qz123456" || stringFromAny(data["panel_password"]) != "Qz123456" {
		t.Fatalf("passwords should not be redacted in downstream response: %#v", data)
	}
	if createHostCount != 1 || hostInfoCount != 1 {
		t.Fatalf("expected create_host=1 hostinfo=1, got %d %d", createHostCount, hostInfoCount)
	}
	if meta, _ := metaRepo.Get(context.Background(), 911418); meta == nil || meta.DataDiskSizeGB != 100 || meta.SysDiskSizeGB != 50 {
		t.Fatalf("metadata not persisted correctly: %#v", meta)
	}
}

func TestServiceOSListSplitsHostInfoAndMirrorImageAndCachesMirror(t *testing.T) {
	var hostInfoCount, mirrorCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/hostinfo"):
			hostInfoCount++
			writeJSON(w, map[string]any{
				"code": 1,
				"msg":  "succ",
				"data": map[string]any{
					"id":      4370,
					"line_id": 7,
				},
			})
		case strings.Contains(r.URL.Path, "/mirror_image"):
			mirrorCount++
			writeJSON(w, map[string]any{
				"code": 1,
				"msg":  "succ",
				"data": []map[string]any{
					{"id": 1, "image_id": 1, "name": "win2022", "type": "windows", "desc": "Windows Server 2022"},
				},
			})
		default:
			t.Fatalf("unexpected upstream path: %s", r.URL.String())
		}
	}))
	defer server.Close()

	svc := NewService(
		upstream.NewClient(server.URL, "apikey", 5*time.Second),
		noopLogRepo{},
		newMemoryMetadataRepo(),
		cacheinfra.NewMemoryCache(),
		30*time.Second,
		1<<20,
		nil,
	)

	call := func(bypass bool) map[string]any {
		resp, err := svc.Handle(context.Background(), RequestContext{
			Method:      http.MethodPost,
			Path:        "/api/v1/osList",
			Body:        []byte(`{"hostid":"4370"}`),
			BypassCache: bypass,
		})
		if err != nil {
			t.Fatalf("Handle osList: %v", err)
		}
		return decodeJSONMap(t, resp.Body)
	}

	first := call(false)
	second := call(false)
	third := call(true)
	for _, payload := range []map[string]any{first, second, third} {
		if intFromAny(payload["code"]) != 0 || stringFromAny(payload["msg"]) != "success" {
			t.Fatalf("unexpected osList envelope: %#v", payload)
		}
		if _, ok := payload["data"].([]any); !ok {
			t.Fatalf("expected osList data array, got %#v", payload["data"])
		}
	}
	if hostInfoCount != 3 {
		t.Fatalf("expected hostinfo 3 times, got %d", hostInfoCount)
	}
	if mirrorCount != 2 {
		t.Fatalf("expected mirror_image 2 times (one cached hit), got %d", mirrorCount)
	}
}

func TestServiceFailuresUseHTTP200BusinessEnvelope(t *testing.T) {
	htmlServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<html><body>error</body></html>"))
	}))
	defer htmlServer.Close()

	svc := NewService(
		upstream.NewClient(htmlServer.URL, "apikey", 5*time.Second),
		noopLogRepo{},
		newMemoryMetadataRepo(),
		cacheinfra.NewMemoryCache(),
		30*time.Second,
		1<<20,
		nil,
	)

	infoResp, err := svc.Handle(context.Background(), RequestContext{
		Method: http.MethodPost,
		Path:   "/api/v1/info",
		Body:   []byte(`{"hostid":"1"}`),
	})
	if err != nil {
		t.Fatalf("Handle info: %v", err)
	}
	infoPayload := decodeJSONMap(t, infoResp.Body)
	if infoResp.StatusCode != 200 || intFromAny(infoPayload["code"]) != 0 {
		t.Fatalf("expected HTTP 200 business failure, got status=%d payload=%#v", infoResp.StatusCode, infoPayload)
	}
	if _, ok := infoPayload["data"].(map[string]any); !ok {
		t.Fatalf("expected info failure data object, got %#v", infoPayload["data"])
	}

	unsupportedResp, err := svc.Handle(context.Background(), RequestContext{
		Method: http.MethodPost,
		Path:   "/api/v1/thumbnail",
		Body:   []byte(`{"hostid":"1"}`),
	})
	if err != nil {
		t.Fatalf("Handle thumbnail: %v", err)
	}
	unsupportedPayload := decodeJSONMap(t, unsupportedResp.Body)
	if unsupportedResp.StatusCode != 200 || intFromAny(unsupportedPayload["code"]) != 0 || stringFromAny(unsupportedPayload["msg"]) != "not implemented" {
		t.Fatalf("unexpected unsupported payload: %#v", unsupportedPayload)
	}
	if _, ok := unsupportedPayload["data"].(map[string]any); !ok {
		t.Fatalf("expected thumbnail failure data object, got %#v", unsupportedPayload["data"])
	}
}

func TestServiceVNCWrapsRedirectURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/vnc_view") {
			t.Fatalf("unexpected upstream path: %s", r.URL.String())
		}
		w.Header().Set("Location", "/console?id=1")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	svc := NewService(
		upstream.NewClient(server.URL, "apikey", 5*time.Second),
		noopLogRepo{},
		newMemoryMetadataRepo(),
		cacheinfra.NewMemoryCache(),
		30*time.Second,
		1<<20,
		nil,
	)

	resp, err := svc.Handle(context.Background(), RequestContext{
		Method: http.MethodPost,
		Path:   "/api/v1/vnc",
		Body:   []byte(`{"hostid":"4349"}`),
	})
	if err != nil {
		t.Fatalf("Handle vnc: %v", err)
	}
	payload := decodeJSONMap(t, resp.Body)
	data := payload["data"].(map[string]any)
	if intFromAny(payload["code"]) != 200 || stringFromAny(payload["msg"]) != "success" {
		t.Fatalf("unexpected vnc envelope: %#v", payload)
	}
	if !strings.Contains(stringFromAny(data["url"]), "/console?id=1") {
		t.Fatalf("expected wrapped redirect url, got %#v", data)
	}
}

func writeJSON(w http.ResponseWriter, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(payload)
}

func decodeJSONMap(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return payload
}
