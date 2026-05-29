package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"xiaoheiproxy/internal/domain"
	"xiaoheiproxy/internal/infrastructure/upstream"

	"gorm.io/gorm"
)

type executionResult struct {
	response        *Response
	primaryRequest  *upstream.Request
	primaryResponse *upstream.Response
	cacheHit        bool
	success         bool
	message         string
}

type cachedUpstreamResponse struct {
	StatusCode int                 `json:"status_code"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
	URL        string              `json:"url"`
	Method     string              `json:"method"`
	RequestHdr map[string][]string `json:"request_headers"`
}

func (s *Service) execute(ctx context.Context, req RequestContext, payload map[string]any) executionResult {
	switch req.Path {
	case "/api/v1/openHost":
		return s.executeOpenHost(ctx, req, payload)
	case "/api/v1/info":
		return s.executeInfo(ctx, req, payload)
	case "/api/v1/osList":
		return s.executeOSList(ctx, req, payload)
	case "/api/v1/vnc", "/api/v1/panel":
		return s.executeRedirect(ctx, req, payload)
	default:
		return s.executeGeneric(ctx, req, payload)
	}
}

func parseRequestPayload(body []byte, query url.Values) map[string]any {
	payload := map[string]any{}
	if len(body) > 0 {
		var obj map[string]any
		if json.Unmarshal(body, &obj) == nil {
			for key, value := range obj {
				payload[key] = value
			}
		} else if values, err := url.ParseQuery(string(body)); err == nil {
			for key, vals := range values {
				if len(vals) > 0 {
					payload[key] = strings.TrimSpace(vals[0])
				}
			}
		}
	}
	for key, vals := range query {
		if len(vals) == 0 {
			continue
		}
		payload[key] = strings.TrimSpace(vals[0])
	}
	return payload
}

func (s *Service) executeGeneric(ctx context.Context, req RequestContext, payload map[string]any) executionResult {
	mapped := MapV2ToV1(req.Path, payload)
	if !mapped.Supported {
		msg := gatewayFailureMessage(req.Path, mapped.Reason)
		return executionResult{
			response: buildFailureResponse(req.Path, msg),
			success:  false,
			message:  msg,
		}
	}

	upResp, cacheHit, err := s.callUpstream(ctx, mapped.Request, "", false, req.BypassCache)
	result := executionResult{
		primaryRequest:  &mapped.Request,
		primaryResponse: upResp,
		cacheHit:        cacheHit,
	}
	if err != nil {
		result.response = buildFailureResponse(req.Path, err.Error())
		result.message = err.Error()
		return result
	}
	resp, success, msg := transformUpstreamResponse(req.Path, payload, upResp)
	result.response = resp
	result.success = success
	result.message = msg
	if success {
		s.afterSuccessfulMutation(ctx, req.Path, payload)
	}
	return result
}

func (s *Service) executeRedirect(ctx context.Context, req RequestContext, payload map[string]any) executionResult {
	mapped := MapV2ToV1(req.Path, payload)
	if !mapped.Supported {
		msg := gatewayFailureMessage(req.Path, mapped.Reason)
		return executionResult{
			response: buildFailureResponse(req.Path, msg),
			success:  false,
			message:  msg,
		}
	}
	upResp, cacheHit, err := s.callUpstream(ctx, mapped.Request, "", false, req.BypassCache)
	result := executionResult{
		primaryRequest:  &mapped.Request,
		primaryResponse: upResp,
		cacheHit:        cacheHit,
	}
	if err != nil {
		result.response = buildFailureResponse(req.Path, err.Error())
		result.message = err.Error()
		return result
	}
	if upResp != nil && upResp.StatusCode >= 300 && upResp.StatusCode < 400 {
		location := strings.TrimSpace(upResp.Headers.Get("Location"))
		if location != "" {
			result.response = buildSuccessResponse(req.Path, map[string]any{"url": resolveRelative(upResp.URL, location)})
			result.success = true
			result.message = contractFor(req.Path).SuccessMsg
			return result
		}
	}
	resp, success, msg := transformUpstreamResponse(req.Path, payload, upResp)
	result.response = resp
	result.success = success
	result.message = msg
	return result
}

func (s *Service) executeInfo(ctx context.Context, req RequestContext, payload map[string]any) executionResult {
	mapped := MapV2ToV1(req.Path, payload)
	if !mapped.Supported {
		msg := gatewayFailureMessage(req.Path, mapped.Reason)
		return executionResult{
			response: buildFailureResponse(req.Path, msg),
			success:  false,
			message:  msg,
		}
	}

	upResp, cacheHit, err := s.callUpstream(ctx, mapped.Request, "", false, req.BypassCache)
	result := executionResult{
		primaryRequest:  &mapped.Request,
		primaryResponse: upResp,
		cacheHit:        cacheHit,
	}
	if err != nil {
		result.response = buildFailureResponse(req.Path, err.Error())
		result.message = err.Error()
		return result
	}

	resp, success, msg := transformUpstreamResponse(req.Path, payload, upResp)
	if !success {
		result.response = resp
		result.success = false
		result.message = msg
		return result
	}

	data := responseDataMap(resp.Body)
	if meta := s.loadMetadata(ctx, uint(firstIntValue(data, "id", "host_id"))); meta != nil {
		data = applyHostMetadata(data, meta)
		resp = buildSuccessResponse(req.Path, data)
	}
	result.response = resp
	result.success = true
	result.message = contractFor(req.Path).SuccessMsg
	return result
}

func (s *Service) executeOpenHost(ctx context.Context, req RequestContext, payload map[string]any) executionResult {
	mapped := MapV2ToV1(req.Path, payload)
	if !mapped.Supported {
		msg := gatewayFailureMessage(req.Path, mapped.Reason)
		return executionResult{
			response: buildFailureResponse(req.Path, msg),
			success:  false,
			message:  msg,
		}
	}

	upResp, cacheHit, err := s.callUpstream(ctx, mapped.Request, "", false, req.BypassCache)
	result := executionResult{
		primaryRequest:  &mapped.Request,
		primaryResponse: upResp,
		cacheHit:        cacheHit,
	}
	if err != nil {
		result.response = buildFailureResponse(req.Path, err.Error())
		result.message = err.Error()
		return result
	}

	resp, success, msg := transformUpstreamResponse(req.Path, payload, upResp)
	if !success {
		result.response = resp
		result.success = false
		result.message = msg
		return result
	}

	data := responseDataMap(resp.Body)
	hostID := uint(firstIntValue(data, "id", "host_id"))
	meta := buildOpenMetadata(payload)
	meta.HostID = hostID
	if hostName := stringValue(data, "host_name"); hostName != "" {
		meta.HostName = hostName
	}
	if hostID != 0 && s.metadata != nil {
		_ = s.metadata.Upsert(ctx, &meta)
	}

	if hostID != 0 {
		infoPayload := map[string]any{"hostid": strconv.Itoa(int(hostID))}
		infoMapped := MapV2ToV1("/api/v1/info", infoPayload)
		infoResp, _, infoErr := s.callUpstream(ctx, infoMapped.Request, "", false, req.BypassCache)
		if infoErr == nil {
			infoJSON, infoSuccess, _ := transformUpstreamResponse("/api/v1/info", infoPayload, infoResp)
			if infoSuccess {
				data = responseDataMap(infoJSON.Body)
			}
		}
	}

	data = applyHostMetadata(data, &meta)
	result.response = buildSuccessResponse(req.Path, data)
	result.success = true
	result.message = contractFor(req.Path).SuccessMsg
	if hostID != 0 && s.metadata != nil {
		meta = metadataFromHostData(meta, data)
		_ = s.metadata.Upsert(ctx, &meta)
	}
	return result
}

func (s *Service) executeOSList(ctx context.Context, req RequestContext, payload map[string]any) executionResult {
	if strings.TrimSpace(stringOf(payload, "hostid")) == "" && strings.TrimSpace(stringOf(payload, "host_id")) == "" {
		msg := "host_id 错误"
		return executionResult{
			response: buildFailureResponse(req.Path, msg),
			success:  false,
			message:  msg,
		}
	}

	infoPayload := map[string]any{"hostid": firstNonBlank(stringOf(payload, "hostid"), stringOf(payload, "host_id"))}
	infoMapped := MapV2ToV1("/api/v1/info", infoPayload)
	infoResp, _, err := s.callUpstream(ctx, infoMapped.Request, "", false, req.BypassCache)
	if err != nil {
		return executionResult{
			response:        buildFailureResponse(req.Path, err.Error()),
			primaryRequest:  &infoMapped.Request,
			primaryResponse: infoResp,
			success:         false,
			message:         err.Error(),
		}
	}
	infoEnv, err := parseUpstreamEnvelope(infoResp.Body)
	if err != nil || !isEnvelopeSuccess("/api/v1/info", infoEnv) {
		msg := "upstream error"
		if err == nil {
			msg = firstNonBlank(infoEnv.Msg, msg)
		}
		return executionResult{
			response:        buildFailureResponse(req.Path, msg),
			primaryRequest:  &infoMapped.Request,
			primaryResponse: infoResp,
			success:         false,
			message:         msg,
		}
	}

	infoData := parseObject(infoEnv.Data)
	lineID := firstIntValue(infoData, "line_id")
	if lineID == 0 {
		meta := s.loadMetadata(ctx, uint(firstIntValue(infoData, "id", "host_id")))
		if meta != nil {
			lineID = meta.LineID
		}
	}
	if lineID == 0 {
		msg := "line_id 缺失"
		return executionResult{
			response:        buildFailureResponse(req.Path, msg),
			primaryRequest:  &infoMapped.Request,
			primaryResponse: infoResp,
			success:         false,
			message:         msg,
		}
	}

	mirrorPayload := map[string]any{"line_id": strconv.Itoa(lineID)}
	mirrorMapped := MapV2ToV1(req.Path, mirrorPayload)
	cacheKey := fmt.Sprintf("static:mirror_image:%d", lineID)
	mirrorResp, cacheHit, mirrorErr := s.callUpstream(ctx, mirrorMapped.Request, cacheKey, true, req.BypassCache)
	result := executionResult{
		primaryRequest:  &mirrorMapped.Request,
		primaryResponse: mirrorResp,
		cacheHit:        cacheHit,
	}
	if mirrorErr != nil {
		result.response = buildFailureResponse(req.Path, mirrorErr.Error())
		result.message = mirrorErr.Error()
		return result
	}

	resp, success, msg := transformUpstreamResponse(req.Path, mirrorPayload, mirrorResp)
	result.response = resp
	result.success = success
	result.message = msg
	return result
}

func (s *Service) callUpstream(ctx context.Context, req upstream.Request, cacheKey string, allowCache bool, bypass bool) (*upstream.Response, bool, error) {
	s.mu.RLock()
	up := s.upstream
	cache := s.cache
	cacheTTL := s.cacheTTL
	s.mu.RUnlock()

	if allowCache && !bypass && cacheKey != "" {
		if raw, ok := cache.Get(cacheKey); ok {
			var cached cachedUpstreamResponse
			if json.Unmarshal(raw, &cached) == nil {
				return &upstream.Response{
					StatusCode: cached.StatusCode,
					Headers:    http.Header(cached.Headers),
					Body:       cached.Body,
					URL:        cached.URL,
					Method:     cached.Method,
					RequestHdr: http.Header(cached.RequestHdr),
				}, true, nil
			}
		}
	}

	resp, err := up.Do(ctx, req)
	if err != nil {
		return nil, false, err
	}

	if allowCache && !bypass && cacheKey != "" && resp.StatusCode >= 200 && resp.StatusCode < 300 && !responseLooksLikeHTML(resp.Body) {
		raw, marshalErr := json.Marshal(cachedUpstreamResponse{
			StatusCode: resp.StatusCode,
			Headers:    map[string][]string(resp.Headers),
			Body:       resp.Body,
			URL:        resp.URL,
			Method:     resp.Method,
			RequestHdr: map[string][]string(resp.RequestHdr),
		})
		if marshalErr == nil {
			cache.Set(cacheKey, raw, cacheTTL)
		}
	}
	return resp, false, nil
}

func buildSuccessResponse(path string, data any) *Response {
	contract := contractFor(path)
	out := map[string]any{
		"msg":  contract.SuccessMsg,
		"code": contract.SuccessCode,
		"time": time.Now().Unix(),
	}
	if contract.HasData {
		if data == nil {
			data = emptyDataForShape(contract.Shape)
		}
		out["data"] = data
	}
	return &Response{
		StatusCode:  http.StatusOK,
		Headers:     jsonHeaders(),
		Body:        mustJSON(out),
		ContentType: "application/json",
	}
}

func buildFailureResponse(path, msg string) *Response {
	contract := contractFor(path)
	out := map[string]any{
		"msg":  firstNonBlank(msg, "upstream error"),
		"code": 0,
		"time": time.Now().Unix(),
	}
	if contract.HasData {
		out["data"] = emptyDataForShape(contract.Shape)
	}
	return &Response{
		StatusCode:  http.StatusOK,
		Headers:     jsonHeaders(),
		Body:        mustJSON(out),
		ContentType: "application/json",
	}
}

func transformUpstreamResponse(path string, payload map[string]any, upResp *upstream.Response) (*Response, bool, string) {
	if upResp == nil {
		msg := "upstream empty response"
		return buildFailureResponse(path, msg), false, msg
	}
	if responseLooksLikeHTML(upResp.Body) {
		msg := "unexpected upstream html response"
		return buildFailureResponse(path, msg), false, msg
	}
	if strings.EqualFold(path, "/api/v1/findport") {
		var findResp struct {
			Code    int     `json:"code"`
			Msg     string  `json:"msg"`
			Content []int64 `json:"content"`
		}
		if err := json.Unmarshal(upResp.Body, &findResp); err == nil {
			success := findResp.Code == 0 || findResp.Code == 1 || findResp.Code == 200
			if success {
				return buildSuccessResponse(path, findResp.Content), true, contractFor(path).SuccessMsg
			}
			msg := firstNonBlank(findResp.Msg, "upstream error")
			return buildFailureResponse(path, msg), false, msg
		}
	}

	env, err := parseUpstreamEnvelope(upResp.Body)
	if err != nil {
		msg := fallbackUpstreamMessage(upResp.StatusCode, upResp.Body)
		return buildFailureResponse(path, msg), false, msg
	}
	if !isEnvelopeSuccess(path, env) {
		msg := firstNonBlank(env.Msg, "upstream error")
		return buildFailureResponse(path, msg), false, msg
	}

	data := transformUpstreamData(path, payload, env.Data)
	return buildSuccessResponse(path, data), true, contractFor(path).SuccessMsg
}

func parseUpstreamEnvelope(body []byte) (apiEnvelope, error) {
	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return apiEnvelope{}, err
	}
	return env, nil
}

func isEnvelopeSuccess(path string, env apiEnvelope) bool {
	if strings.EqualFold(path, "/api/v1/findport") {
		return env.Code == 0 || env.Code == 1 || env.Code == 200
	}
	return env.Code == 1 || env.Code == 200
}

func transformUpstreamData(path string, payload map[string]any, raw json.RawMessage) any {
	switch path {
	case "/api/v1/openHost", "/api/v1/info":
		return normalizeHostPayload(raw)
	case "/api/v1/osList":
		return normalizeImagePayload(raw)
	case "/api/v1/monitor":
		return normalizeMonitorPayload(raw)
	case "/api/v1/snapshot", "/api/v1/backup":
		return normalizeCollectionWithExtend(raw, payload)
	case "/api/v1/createSnapshot":
		return normalizeCollectionWithExtend(raw, payload)
	case "/api/v1/firewallList":
		return normalizeFirewallList(raw)
	case "/api/v1/addFirewall", "/api/v1/removeFirewall":
		return normalizeFirewallList(raw)
	case "/api/v1/portList":
		return normalizePortList(raw)
	case "/api/v1/removeSnapshot", "/api/v1/restoreSnapshot",
		"/api/v1/createBackup", "/api/v1/removeBackup", "/api/v1/restoreBackupHost",
		"/api/v1/addPort", "/api/v1/removePort",
		"/api/v1/addDomain", "/api/v1/removeDomain":
		return []any{}
	default:
		contract := contractFor(path)
		if !contract.HasData {
			return nil
		}
		switch contract.Shape {
		case DataShapeArray:
			var items []any
			if json.Unmarshal(raw, &items) == nil {
				return items
			}
			return []any{}
		case DataShapeObject, DataShapeURL, DataShapeDataExtend:
			var obj map[string]any
			if json.Unmarshal(raw, &obj) == nil {
				return obj
			}
			return emptyDataForShape(contract.Shape)
		default:
			return nil
		}
	}
}

func fallbackUpstreamMessage(status int, body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		if status >= 400 {
			return fmt.Sprintf("upstream http %d", status)
		}
		return "upstream response parse failed"
	}
	if len(text) > 240 {
		text = text[:240]
	}
	return text
}

func gatewayFailureMessage(path, reason string) string {
	reason = strings.TrimSpace(reason)
	switch {
	case reason == "":
		return "not implemented"
	case strings.Contains(reason, "host_id"):
		return reason
	case strings.Contains(reason, "line_id"):
		return reason
	case strings.Contains(reason, "not audited"):
		return "not implemented"
	case strings.Contains(reason, "no public api implementation"):
		return "not implemented"
	default:
		if isKnownEndpoint(path) {
			return reason
		}
		return "not implemented"
	}
}

func responseDataMap(body []byte) map[string]any {
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return map[string]any{}
	}
	if data, ok := payload["data"].(map[string]any); ok {
		return data
	}
	return map[string]any{}
}

func (s *Service) loadMetadata(ctx context.Context, hostID uint) *domain.HostV2Metadata {
	if s.metadata == nil || hostID == 0 {
		return nil
	}
	metadata, err := s.metadata.Get(ctx, hostID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return nil
	}
	return metadata
}

func (s *Service) afterSuccessfulMutation(ctx context.Context, path string, payload map[string]any) {
	if s.metadata == nil {
		return
	}
	hostID := uint(intFromPayload(payload, "hostid", "host_id"))
	if hostID == 0 && path != "/api/v1/panel" && path != "/api/v1/test" {
		return
	}
	switch path {
	case "/api/v1/updateHost":
		meta := s.loadMetadata(ctx, hostID)
		if meta == nil {
			meta = &domain.HostV2Metadata{HostID: hostID}
		}
		mergeUpdateMetadata(meta, payload)
		_ = s.metadata.Upsert(ctx, meta)
	case "/api/v1/renew":
		meta := s.loadMetadata(ctx, hostID)
		if meta == nil {
			meta = &domain.HostV2Metadata{HostID: hostID}
		}
		if _, ok := payload["nextduedate"]; ok {
			meta.ExpireTime = stringOf(payload, "nextduedate")
		}
		_ = s.metadata.Upsert(ctx, meta)
	case "/api/v1/installOS":
		meta := s.loadMetadata(ctx, hostID)
		if meta == nil {
			meta = &domain.HostV2Metadata{HostID: hostID}
		}
		template := strings.TrimSpace(stringOf(payload, "template"))
		if template != "" && !looksNumeric(template) {
			meta.OSName = template
		}
		_ = s.metadata.Upsert(ctx, meta)
	case "/api/v1/removeHost":
		_ = s.metadata.Delete(ctx, hostID)
	}
}

func buildOpenMetadata(payload map[string]any) domain.HostV2Metadata {
	metadata := domain.HostV2Metadata{
		LineID:            intFromPayload(payload, "line_id"),
		NodesID:           intFromPayload(payload, "nodes_id"),
		HostName:          stringOf(payload, "host_name"),
		OSName:            stringOf(payload, "os_name"),
		CPU:               intFromPayload(payload, "cpu"),
		CPULimit:          intFromPayload(payload, "cpu_limit"),
		MemoryMB:          intFromPayload(payload, "memory"),
		SysDiskSizeGB:     intFromPayload(payload, "sys_disk_size"),
		DataDiskSizeGB:    intFromPayload(payload, "data_disk_size"),
		SysDiskIOPS:       intFromPayload(payload, "sys_disk_iops"),
		DataDiskIOPS:      intFromPayload(payload, "data_disk_iops"),
		NetOutMbps:        intFromPayload(payload, "net_out"),
		NetInMbps:         intFromPayload(payload, "net_in"),
		FlowLimitGB:       intFromPayload(payload, "flow_limit"),
		IPNum:             intFromPayload(payload, "ip_num"),
		IsNAT:             intFromPayload(payload, "is_nat"),
		PortNum:           intFromPayload(payload, "port_num"),
		DomainNum:         intFromPayload(payload, "domain_num"),
		SnapshotNum:       intFromPayload(payload, "snapshot"),
		BackupNum:         intFromPayload(payload, "backups"),
		MaxReinstallNum:   intFromPayload(payload, "max_reinstall_num"),
		BuyTime:           stringOf(payload, "buy_time"),
		ExpireTime:        stringOf(payload, "expire_time"),
		LastOpenRequest:   payloadJSON(payload),
		LastUpdateRequest: "",
	}
	return metadata
}

func mergeUpdateMetadata(metadata *domain.HostV2Metadata, payload map[string]any) {
	if _, ok := payload["line_id"]; ok {
		metadata.LineID = intFromPayload(payload, "line_id")
	}
	if _, ok := payload["nodes_id"]; ok {
		metadata.NodesID = intFromPayload(payload, "nodes_id")
	}
	if _, ok := payload["host_name"]; ok {
		metadata.HostName = stringOf(payload, "host_name")
	}
	if _, ok := payload["os_name"]; ok {
		metadata.OSName = stringOf(payload, "os_name")
	}
	if _, ok := payload["cpu"]; ok {
		metadata.CPU = intFromPayload(payload, "cpu")
	}
	if _, ok := payload["cpu_limit"]; ok {
		metadata.CPULimit = intFromPayload(payload, "cpu_limit")
	}
	if _, ok := payload["memory"]; ok {
		metadata.MemoryMB = intFromPayload(payload, "memory")
	}
	if _, ok := payload["sys_disk_size"]; ok {
		metadata.SysDiskSizeGB = intFromPayload(payload, "sys_disk_size")
	}
	if _, ok := payload["data_disk_size"]; ok {
		metadata.DataDiskSizeGB = intFromPayload(payload, "data_disk_size")
	}
	if _, ok := payload["sys_disk_iops"]; ok {
		metadata.SysDiskIOPS = intFromPayload(payload, "sys_disk_iops")
	}
	if _, ok := payload["data_disk_iops"]; ok {
		metadata.DataDiskIOPS = intFromPayload(payload, "data_disk_iops")
	}
	if _, ok := payload["net_out"]; ok {
		metadata.NetOutMbps = intFromPayload(payload, "net_out")
	}
	if _, ok := payload["net_in"]; ok {
		metadata.NetInMbps = intFromPayload(payload, "net_in")
	}
	if _, ok := payload["flow_limit"]; ok {
		metadata.FlowLimitGB = intFromPayload(payload, "flow_limit")
	}
	if _, ok := payload["ip_num"]; ok {
		metadata.IPNum = intFromPayload(payload, "ip_num")
	}
	if _, ok := payload["is_nat"]; ok {
		metadata.IsNAT = intFromPayload(payload, "is_nat")
	}
	if _, ok := payload["port_num"]; ok {
		metadata.PortNum = intFromPayload(payload, "port_num")
	}
	if _, ok := payload["domain_num"]; ok {
		metadata.DomainNum = intFromPayload(payload, "domain_num")
	}
	if _, ok := payload["snapshot"]; ok {
		metadata.SnapshotNum = intFromPayload(payload, "snapshot")
	}
	if _, ok := payload["backups"]; ok {
		metadata.BackupNum = intFromPayload(payload, "backups")
	}
	if _, ok := payload["max_reinstall_num"]; ok {
		metadata.MaxReinstallNum = intFromPayload(payload, "max_reinstall_num")
	}
	if _, ok := payload["buy_time"]; ok {
		metadata.BuyTime = stringOf(payload, "buy_time")
	}
	if _, ok := payload["expire_time"]; ok {
		metadata.ExpireTime = stringOf(payload, "expire_time")
	}
	metadata.LastUpdateRequest = payloadJSON(payload)
}

func metadataFromHostData(metadata domain.HostV2Metadata, data map[string]any) domain.HostV2Metadata {
	if metadata.HostName == "" {
		metadata.HostName = stringValue(data, "host_name")
	}
	if metadata.BuyTime == "" {
		metadata.BuyTime = stringValue(data, "buy_time")
	}
	if metadata.ExpireTime == "" {
		metadata.ExpireTime = stringValue(data, "end_time")
	}
	return metadata
}

func applyHostMetadata(host map[string]any, metadata *domain.HostV2Metadata) map[string]any {
	if host == nil {
		host = map[string]any{}
	}
	if metadata == nil {
		return host
	}
	if metadata.HostID != 0 {
		host["id"] = int(metadata.HostID)
	}
	if metadata.HostName != "" {
		host["host_name"] = metadata.HostName
	}
	if metadata.OSName != "" {
		host["os_name"] = metadata.OSName
	}
	if metadata.CPU > 0 {
		host["cpu"] = metadata.CPU
	}
	host["cpu_limit"] = metadata.CPULimit
	if metadata.MemoryMB > 0 {
		host["memory"] = normalizeMemoryResponse(metadata.MemoryMB)
	}
	if metadata.DataDiskSizeGB > 0 {
		host["hard_disks"] = metadata.DataDiskSizeGB
	}
	if metadata.SysDiskSizeGB > 0 {
		host["os_size"] = metadata.SysDiskSizeGB
	}
	host["bandwidth_out"] = metadata.NetOutMbps
	host["bandwidth_in"] = metadata.NetInMbps
	host["traffic"] = metadata.FlowLimitGB
	host["is_nat"] = metadata.IsNAT
	host["port_num"] = metadata.PortNum
	host["domain_num"] = metadata.DomainNum
	host["snapshot_num"] = metadata.SnapshotNum
	host["backup_num"] = metadata.BackupNum
	host["max_reinstall_num"] = metadata.MaxReinstallNum
	if metadata.BuyTime != "" {
		host["buy_time"] = metadata.BuyTime
	}
	if metadata.ExpireTime != "" {
		host["end_time"] = metadata.ExpireTime
	}
	return host
}

func normalizeMemoryResponse(memoryMB int) int {
	if memoryMB >= 1024 && memoryMB%1024 == 0 {
		return memoryMB / 1024
	}
	return memoryMB
}

func intFromPayload(payload map[string]any, keys ...string) int {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		switch t := value.(type) {
		case int:
			return t
		case int32:
			return int(t)
		case int64:
			return int(t)
		case float64:
			return int(t)
		case float32:
			return int(t)
		case json.Number:
			if i, err := t.Int64(); err == nil {
				return int(i)
			}
		case string:
			if i, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
				return i
			}
		}
	}
	return 0
}

func payloadJSON(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(b)
}

func looksNumeric(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	_, err := strconv.Atoi(value)
	return err == nil
}
