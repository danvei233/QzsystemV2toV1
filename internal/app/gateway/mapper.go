package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"xiaoheiproxy/internal/infrastructure/upstream"
)

type MappingResult struct {
	Request   upstream.Request
	Cacheable bool
	Supported bool
}

func MapV2ToV1(path string, body []byte) MappingResult {
	payload := map[string]any{}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &payload)
	}
	action := strings.TrimPrefix(path, "/api/v1/")
	values := url.Values{}
	contentType := "application/x-www-form-urlencoded"
	setHostID := func() {
		hostID := stringOf(payload, "hostid")
		if hostID == "" {
			hostID = stringOf(payload, "host_id")
		}
		if hostID != "" {
			values.Set("host_id", hostID)
			values.Set("hostid", hostID)
		}
	}
	setIf := func(from, to string) {
		if v := stringOf(payload, from); v != "" {
			values.Set(to, v)
		}
	}
	setAll := func() {
		for key := range payload {
			if v := stringOf(payload, key); v != "" {
				values.Set(key, v)
			}
		}
	}
	formReq := func(method, upstreamPath string) MappingResult {
		return MappingResult{
			Request: upstream.Request{
				Method:      method,
				Path:        upstreamPath,
				Body:        []byte(values.Encode()),
				ContentType: contentType,
			},
			Supported: true,
		}
	}
	queryReq := func(method, upstreamPath string) MappingResult {
		return MappingResult{
			Request:   upstream.Request{Method: method, Path: upstreamPath, Query: values},
			Supported: true,
		}
	}

	switch action {
	case "openHost":
		setIf("line_id", "line_id")
		setIf("nodes_id", "nodes_id")
		setIf("os_name", "os")
		setIf("cpu", "cpu")
		setIf("memory", "memory")
		setIf("sys_disk_size", "hard_disks")
		setIf("data_disk_size", "data_disk_size")
		if v := first(payload, "net_out", "net_in"); v != "" {
			values.Set("bandwidth", v)
		}
		setIf("expire_time", "expire_time")
		setIf("host_name", "host_name")
		setIf("sys_pwd", "sys_pwd")
		setIf("snapshot", "snapshot")
		setIf("backups", "backups")
		setIf("port_num", "port_num")
		setIf("domain_num", "domain_num")
		return queryReq(http.MethodPost, "create_host")
	case "updateHost":
		setHostID()
		setIf("cpu", "cpu")
		setIf("memory", "memory")
		setIf("sys_disk_size", "hard_disks")
		setIf("data_disk_size", "data_disk_size")
		if v := first(payload, "net_out", "net_in"); v != "" {
			values.Set("bandwidth", v)
		}
		setIf("port_num", "port_num")
		return queryReq(http.MethodPost, "elastic_update")
	case "removeHost":
		setHostID()
		return formReq(http.MethodPost, "delete")
	case "info":
		setHostID()
		return cacheable(formReq(http.MethodPost, "hostinfo"))
	case "renew":
		setHostID()
		setIf("nextduedate", "nextduedate")
		return formReq(http.MethodPost, "renew")
	case "power":
		setHostID()
		switch strings.TrimSpace(stringOf(payload, "state")) {
		case "2", "start", "boot", "on":
			return formReq(http.MethodPost, "start")
		case "5", "reboot", "restart":
			return formReq(http.MethodPost, "reboot")
		default:
			return formReq(http.MethodPost, "shutdown")
		}
	case "monitor":
		setHostID()
		return cacheable(formReq(http.MethodPost, "monitor"))
	case "updateOSPassword":
		setHostID()
		setIf("password", "password")
		return formReq(http.MethodPost, "reset_password")
	case "osList":
		setIf("line_id", "line_id")
		return cacheable(queryReq(http.MethodGet, "mirror_image"))
	case "installOS":
		setHostID()
		setIf("template", "template_id")
		setIf("password", "password")
		return formReq(http.MethodPost, "reset_os")
	case "snapshot":
		setHostID()
		return cacheable(formReq(http.MethodPost, "snapshot_list"))
	case "createSnapshot":
		setHostID()
		return formReq(http.MethodPost, "snapshot_add")
	case "removeSnapshot":
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "snapshot_del")
	case "restoreSnapshot":
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "snapshot_restore")
	case "backup":
		setHostID()
		return cacheable(formReq(http.MethodPost, "backups_list"))
	case "createBackup":
		setHostID()
		return formReq(http.MethodPost, "backups_add")
	case "removeBackup":
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "backups_del")
	case "restoreBackupHost":
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "backups_restore")
	case "firewallList":
		setHostID()
		setIf("page", "page")
		setIf("direction", "direction")
		setIf("method", "method")
		setIf("protocol", "protocol")
		return cacheable(queryReq(http.MethodGet, "security_acl_list"))
	case "addFirewall":
		setHostID()
		setIf("direction", "direction")
		setIf("method", "method")
		setIf("protocol", "protocol")
		setIf("port", "port")
		setIf("ip", "ip")
		setIf("priority", "priority")
		setIf("remark", "remark")
		return formReq(http.MethodPost, "security_acl_add")
	case "removeFirewall":
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "security_acl_del")
	case "portList":
		setHostID()
		return cacheable(queryReq(http.MethodGet, "nat_acl_list"))
	case "addPort":
		setHostID()
		setIf("dport", "dport")
		setIf("sport", "sport")
		setIf("name", "name")
		return formReq(http.MethodPost, "add_port_host")
	case "removePort":
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "remove_port_host")
	case "findport":
		setHostID()
		setIf("keywords", "keywords")
		return cacheable(queryReq(http.MethodGet, "findport"))
	case "vnc":
		setHostID()
		return formReq(http.MethodPost, "vnc_view")
	case "panel":
		setAll()
		return formReq(http.MethodPost, "panel")
	case "test":
		setAll()
		return formReq(http.MethodPost, "test")
	default:
		setAll()
		return formReq(http.MethodPost, action)
	}
}

func cacheable(result MappingResult) MappingResult {
	result.Cacheable = true
	return result
}

func first(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if v := stringOf(payload, key); v != "" {
			return v
		}
	}
	return ""
}

func stringOf(payload map[string]any, key string) string {
	v, ok := payload[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return t.String()
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}
