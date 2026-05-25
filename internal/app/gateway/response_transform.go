package gateway

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func isUpstreamSuccess(path string, status int, body []byte) bool {
	if status < 200 || status >= 400 {
		return false
	}
	if strings.EqualFold(path, "/api/v1/findport") {
		var resp struct {
			Code int `json:"code"`
		}
		if json.Unmarshal(body, &resp) == nil {
			return resp.Code == 0 || resp.Code == 1 || resp.Code == 200
		}
	}
	var env apiEnvelope
	if json.Unmarshal(body, &env) == nil {
		return env.Code == 1 || env.Code == 200
	}
	return true
}

func normalizeEnvelopeBody(path string, requestBody []byte, body []byte, status int) []byte {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || status >= 300 {
		return nil
	}

	if strings.EqualFold(path, "/api/v1/findport") {
		var findResp struct {
			Code    int     `json:"code"`
			Msg     string  `json:"msg"`
			Content []int64 `json:"content"`
		}
		if json.Unmarshal(body, &findResp) == nil {
			return mustJSON(map[string]any{
				"msg":  firstNonBlank(findResp.Msg, successMessage(path)),
				"code": successCode(path, findResp.Code == 0 || findResp.Code == 1 || findResp.Code == 200),
				"time": time.Now().Unix(),
				"data": findResp.Content,
			})
		}
	}

	var env apiEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil
	}
	success := env.Code == 1 || env.Code == 200
	if !success {
		out := map[string]any{
			"msg":  firstNonBlank(env.Msg, "upstream error"),
			"code": env.Code,
			"time": nonZeroTime(env.Time),
		}
		if len(env.Data) > 0 {
			var raw any
			if json.Unmarshal(env.Data, &raw) == nil {
				out["data"] = raw
			}
		}
		return mustJSON(out)
	}

	switch path {
	case "/api/v1/openHost", "/api/v1/info":
		return buildSuccessEnvelope(path, normalizeHostPayload(env.Data), success)
	case "/api/v1/updateHost", "/api/v1/removeHost", "/api/v1/renew", "/api/v1/power", "/api/v1/updateOSPassword", "/api/v1/installOS":
		return buildSimpleSuccessEnvelope(path, env.Msg, success)
	case "/api/v1/osList":
		return buildSuccessEnvelope(path, normalizeImagePayload(env.Data), success)
	case "/api/v1/monitor":
		return buildSuccessEnvelope(path, normalizeMonitorPayload(env.Data), success)
	case "/api/v1/snapshot", "/api/v1/backup":
		return buildSuccessEnvelope(path, normalizeCollectionWithExtend(env.Data, requestBody, path), success)
	case "/api/v1/firewallList":
		return buildSuccessEnvelope(path, normalizeFirewallList(env.Data), success)
	case "/api/v1/portList":
		return buildSuccessEnvelope(path, normalizePortList(env.Data), success)
	case "/api/v1/createSnapshot", "/api/v1/removeSnapshot", "/api/v1/restoreSnapshot",
		"/api/v1/createBackup", "/api/v1/removeBackup", "/api/v1/restoreBackupHost",
		"/api/v1/addPort", "/api/v1/removePort",
		"/api/v1/addFirewall", "/api/v1/removeFirewall":
		return buildSuccessEnvelope(path, []any{}, success)
	case "/api/v1/vnc", "/api/v1/panel":
		// Redirects are normalized before this helper runs.
		return nil
	default:
		var raw any
		if json.Unmarshal(env.Data, &raw) == nil {
			return buildSuccessEnvelope(path, raw, success)
		}
		return buildSimpleSuccessEnvelope(path, env.Msg, success)
	}
}

func buildSimpleSuccessEnvelope(path, msg string, success bool) []byte {
	return mustJSON(map[string]any{
		"msg":  firstNonBlank(msg, successMessage(path)),
		"code": successCode(path, success),
		"time": time.Now().Unix(),
	})
}

func buildSuccessEnvelope(path string, data any, success bool) []byte {
	return mustJSON(map[string]any{
		"msg":  successMessage(path),
		"code": successCode(path, success),
		"time": time.Now().Unix(),
		"data": data,
	})
}

func successCode(path string, success bool) int {
	if !success {
		return 0
	}
	switch path {
	case "/api/v1/openHost", "/api/v1/info":
		return 200
	case "/api/v1/updateHost", "/api/v1/removeHost", "/api/v1/renew", "/api/v1/osList":
		return 0
	default:
		return 200
	}
}

func successMessage(path string) string {
	switch path {
	case "/api/v1/openHost":
		return "success"
	case "/api/v1/updateHost":
		return "success"
	case "/api/v1/removeHost":
		return "success"
	case "/api/v1/info":
		return "success"
	case "/api/v1/renew":
		return "success"
	case "/api/v1/power":
		return "启动命令执行成功"
	case "/api/v1/monitor":
		return "success"
	case "/api/v1/updateOSPassword":
		return "success"
	case "/api/v1/osList":
		return "success"
	case "/api/v1/installOS":
		return "执行重装系统命令成功"
	case "/api/v1/createSnapshot":
		return "执行创建快照命令成功"
	case "/api/v1/removeSnapshot":
		return "执行删除快照命令成功"
	case "/api/v1/restoreSnapshot":
		return "执行恢复快照命令成功"
	case "/api/v1/backup", "/api/v1/snapshot", "/api/v1/firewallList", "/api/v1/portList", "/api/v1/findport", "/api/v1/vnc", "/api/v1/panel":
		return "success"
	case "/api/v1/createBackup":
		return "执行创建备份命令成功"
	case "/api/v1/removeBackup":
		return "执行删除备份命令成功"
	case "/api/v1/restoreBackupHost":
		return "执行恢复备份命令成功"
	case "/api/v1/addPort":
		return "添加端口成功"
	case "/api/v1/removePort":
		return "删除端口成功"
	case "/api/v1/addFirewall":
		return "添加策略成功"
	case "/api/v1/removeFirewall":
		return "删除策略成功"
	default:
		return "success"
	}
}

func normalizeHostPayload(raw json.RawMessage) map[string]any {
	data := parseObject(raw)
	out := map[string]any{
		"id":                intValue(data, "id", "host_id"),
		"area_id":           fmt.Sprint(anyValue(data, "", "area_id")),
		"area_name":         stringValue(data, "area_name"),
		"node_id":           intValue(data, "node_id"),
		"host_name":         stringValue(data, "host_name"),
		"ip":                stringValue(data, "ip"),
		"local_ip":          stringValue(data, "local_ip"),
		"buy_time":          stringValue(data, "buy_time"),
		"end_time":          stringValue(data, "end_time"),
		"cpu":               intValue(data, "cpu"),
		"memory":            intValue(data, "memory"),
		"hard_disks":        intValue(data, "hard_disks"),
		"bandwidth_out":     firstIntValue(data, "bandwidth_out", "bandwidth"),
		"bandwidth_in":      firstIntValue(data, "bandwidth_in", "bandwidth"),
		"os_size":           intValue(data, "os_size"),
		"os_name":           stringValue(data, "os_name"),
		"os_username":       guessOSUsername(stringValue(data, "os_username"), stringValue(data, "os_name")),
		"os_password":       stringValue(data, "os_password"),
		"panel_password":    stringValue(data, "panel_password"),
		"cpu_limit":         intValue(data, "cpu_limit"),
		"state":             intValue(data, "state"),
		"state_name":        firstNonBlank(stringValue(data, "state_name"), hostStateName(intValue(data, "state"))),
		"vlanid1":           intValue(data, "vlanid1"),
		"vlanid2":           intValue(data, "vlanid2"),
		"close_network":     firstNonBlank(stringValue(data, "close_network"), "1"),
		"traffic":           intValue(data, "traffic"),
		"max_reinstall_num": intValue(data, "max_reinstall_num"),
		"reinstall_num":     intValue(data, "reinstall_num"),
		"os_disk_iops":      stringValue(data, "os_disk_iops", "os_disk_maxiops"),
		"data_disk_iops":    intValue(data, "data_disk_iops"),
		"sync_time":         intValue(data, "sync_time"),
		"now_iso":           stringValue(data, "now_iso"),
		"snapshot_num":      intValue(data, "snapshot_num"),
		"backup_num":        intValue(data, "backup_num"),
		"domain_num":        intValue(data, "domain_num"),
		"bios":              stringValue(data, "bios"),
		"metal":             intValue(data, "metal"),
		"is_nat":            intValue(data, "is_nat"),
		"port_num":          intValue(data, "port_num"),
		"mac":               stringValue(data, "mac"),
		"mac1":              stringValue(data, "mac1"),
	}

	publicIPs := extractPublicIPs(data)
	out["allip"] = publicIPs
	if len(publicIPs) > 1 {
		out["attachip"] = publicIPs[1:]
	} else {
		out["attachip"] = []map[string]any{}
	}
	remoteAddr, remotePort := splitRemote(stringValue(data, "remote_addr", "remote_ip"))
	out["remote_addr"] = remoteAddr
	out["remote_port"] = remotePort
	return out
}

func normalizeImagePayload(raw json.RawMessage) []map[string]any {
	items := parseArray(raw)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"id":       firstIntValue(item, "id", "image_id"),
			"image_id": firstIntValue(item, "image_id", "id"),
			"name":     stringValue(item, "name", "os_name"),
			"type":     stringValue(item, "type"),
			"desc":     stringValue(item, "desc", "remark"),
		})
	}
	return out
}

func normalizeMonitorPayload(raw json.RawMessage) map[string]any {
	data := parseObject(raw)
	cpu := floatValue(data, "cpu", "CpuStats")
	memory := intValue(data, "memory", "MemoryStats")
	bytesOut := int64(0)
	bytesIn := int64(0)
	if network, ok := data["NetworkStats"]; ok {
		netObj := normalizeAnyMap(network)
		bytesOut = int64(firstIntValue(netObj, "BytesSentPersec"))
		bytesIn = int64(firstIntValue(netObj, "BytesReceivedPersec"))
	}
	trafficOut := 0
	trafficIn := 0
	if traffic, ok := data["Traffic"]; ok {
		trafficObj := normalizeAnyMap(traffic)
		trafficOut = firstIntValue(trafficObj, "TrafficOut")
		trafficIn = firstIntValue(trafficObj, "TrafficIn")
	}
	return map[string]any{
		"cpu":        cpu,
		"memory":     memory,
		"bwOut":      formatMbps(bytesOut),
		"bwIn":       formatMbps(bytesIn),
		"trafficOut": trafficOut,
		"trafficIn":  trafficIn,
	}
}

func normalizeCollectionWithExtend(raw json.RawMessage, requestBody []byte, path string) map[string]any {
	items := parseArray(raw)
	req := parseObject(requestBody)
	hostID := intValue(req, "hostid", "host_id")
	outItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		hostIDItem := firstIntValue(item, "host_id", "virtuals_id")
		if hostIDItem == 0 {
			hostIDItem = hostID
		}
		createdAt := firstNonBlank(stringValue(item, "create_time", "created_at"), time.Now().Format("2006-01-02 15:04:05"))
		outItems = append(outItems, map[string]any{
			"id":          firstIntValue(item, "id"),
			"name":        stringValue(item, "name"),
			"host_name":   stringValue(item, "host_name"),
			"host_id":     hostIDItem,
			"current":     firstIntValue(item, "current"),
			"state":       firstIntValue(item, "state"),
			"create_time": createdAt,
			"update_time": firstNonBlank(stringValue(item, "update_time", "created_at", "create_time"), createdAt),
		})
		if outItems[len(outItems)-1]["current"].(int) == 0 {
			outItems[len(outItems)-1]["current"] = 1
		}
		if outItems[len(outItems)-1]["state"].(int) == 0 {
			outItems[len(outItems)-1]["state"] = 2
		}
	}
	return map[string]any{
		"data":   outItems,
		"extend": map[string]any{"count": len(outItems)},
	}
}

func normalizeFirewallList(raw json.RawMessage) map[string]any {
	items := parseArray(raw)
	outItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		direction := stringValue(item, "direction")
		method := stringValue(item, "method")
		outItems = append(outItems, map[string]any{
			"id":             firstIntValue(item, "id"),
			"name":           firstNonBlank(stringValue(item, "name"), fmt.Sprintf("%d", firstIntValue(item, "id"))),
			"direction":      direction,
			"direction_name": firstNonBlank(stringValue(item, "direction_name"), directionName(direction)),
			"method":         method,
			"method_name":    firstNonBlank(stringValue(item, "method_name"), methodName(method)),
			"protocol":       stringValue(item, "protocol"),
			"port":           firstNonBlank(stringValue(item, "port"), "ANY"),
			"ip":             firstNonBlank(stringValue(item, "ip"), "ANY"),
			"priority":       firstNonBlank(stringValue(item, "priority"), "1"),
			"remark":         stringValue(item, "remark"),
		})
	}
	return map[string]any{
		"data":   outItems,
		"extend": map[string]any{"count": len(outItems)},
	}
}

func normalizePortList(raw json.RawMessage) []map[string]any {
	items := parseArray(raw)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"id":        firstIntValue(item, "id"),
			"host_id":   firstIntValue(item, "host_id"),
			"host_name": stringValue(item, "host_name"),
			"name":      stringValue(item, "name"),
			"port_type": firstNonBlank(stringValue(item, "port_type"), "all"),
			"sport":     stringValue(item, "sport"),
			"dport":     stringValue(item, "dport"),
			"api_url":   firstNonBlank(stringValue(item, "api_url"), stringValue(item, "public_ip")),
			"sys":       firstIntValue(item, "sys"),
			"dip":       stringValue(item, "dip"),
		})
	}
	return out
}

func parseObject(raw []byte) map[string]any {
	var data map[string]any
	_ = json.Unmarshal(raw, &data)
	if data == nil {
		return map[string]any{}
	}
	return data
}

func parseArray(raw []byte) []map[string]any {
	var items []map[string]any
	if json.Unmarshal(raw, &items) == nil {
		return items
	}
	var wrapper map[string]any
	if json.Unmarshal(raw, &wrapper) == nil {
		if data, ok := wrapper["data"].([]any); ok {
			out := make([]map[string]any, 0, len(data))
			for _, item := range data {
				out = append(out, normalizeAnyMap(item))
			}
			return out
		}
	}
	return []map[string]any{}
}

func normalizeAnyMap(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		return t
	case []byte:
		return parseObject(t)
	case string:
		return parseObject([]byte(t))
	default:
		return map[string]any{}
	}
}

func stringValue(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok && value != nil {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func intValue(data map[string]any, keys ...string) int {
	return firstIntValue(data, keys...)
}

func firstIntValue(data map[string]any, keys ...string) int {
	for _, key := range keys {
		if value, ok := data[key]; ok && value != nil {
			switch t := value.(type) {
			case float64:
				return int(t)
			case float32:
				return int(t)
			case int:
				return t
			case int32:
				return int(t)
			case int64:
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
	}
	return 0
}

func floatValue(data map[string]any, keys ...string) float64 {
	for _, key := range keys {
		if value, ok := data[key]; ok && value != nil {
			switch t := value.(type) {
			case float64:
				return t
			case float32:
				return float64(t)
			case int:
				return float64(t)
			case int32:
				return float64(t)
			case int64:
				return float64(t)
			case json.Number:
				if f, err := t.Float64(); err == nil {
					return f
				}
			case string:
				if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

func anyValue(data map[string]any, fallback string, keys ...string) any {
	for _, key := range keys {
		if value, ok := data[key]; ok && value != nil {
			return value
		}
	}
	return fallback
}

func extractPublicIPs(data map[string]any) []map[string]any {
	network := normalizeAnyMap(data["network"])
	publicList, ok := network["eth1"].([]any)
	if ok {
		out := make([]map[string]any, 0, len(publicList))
		for _, item := range publicList {
			entry := normalizeAnyMap(item)
			if len(entry) > 0 {
				out = append(out, entry)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	primaryIP := stringValue(data, "ip")
	if primaryIP == "" {
		return []map[string]any{}
	}
	all := []map[string]any{{"ip": primaryIP}}
	for _, extra := range strings.Split(stringValue(data, "add_ip"), ",") {
		extra = strings.TrimSpace(extra)
		if extra == "" {
			continue
		}
		all = append(all, map[string]any{"ip": extra})
	}
	return all
}

func splitRemote(raw string) (string, any) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	if idx := strings.LastIndex(raw, ":"); idx > 0 && idx < len(raw)-1 {
		addr := raw[:idx]
		port := raw[idx+1:]
		if _, err := strconv.Atoi(port); err == nil {
			return addr, port
		}
	}
	return raw, ""
}

func hostStateName(state int) string {
	switch state {
	case 1:
		return "创建中"
	case 2:
		return "运行中"
	case 3:
		return "关机"
	case 4:
		return "重装系统中"
	case 5:
		return "重装系统失败"
	case 10:
		return "锁定"
	case 11:
		return "创建失败"
	case 12:
		return "删除中"
	case 13:
		return "重新创建中"
	default:
		return ""
	}
}

func guessOSUsername(current, osName string) string {
	if strings.TrimSpace(current) != "" {
		return current
	}
	if strings.Contains(strings.ToLower(osName), "win") {
		return "Administrator"
	}
	return "root"
}

func formatMbps(bytesPerSec int64) string {
	mbps := float64(bytesPerSec) * 8 / 1024 / 1024
	return fmt.Sprintf("%.2f", mbps)
}

func directionName(direction string) string {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "in":
		return "入"
	case "out":
		return "出"
	default:
		return direction
	}
}

func methodName(method string) string {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "accept":
		return "接受"
	case "drop":
		return "拒绝"
	case "reject":
		return "拒绝"
	default:
		return method
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func nonZeroTime(v int64) int64 {
	if v > 0 {
		return v
	}
	return time.Now().Unix()
}
