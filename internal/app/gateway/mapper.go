package gateway

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"xiaoheiproxy/internal/infrastructure/upstream"
)

type MappingResult struct {
	Request   upstream.Request
	Supported bool
	Reason    string
}

func MapV2ToV1(path string, payload map[string]any) MappingResult {
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
	requireHostID := func() MappingResult {
		hostID := stringOf(payload, "hostid")
		if hostID == "" {
			hostID = stringOf(payload, "host_id")
		}
		if hostID == "" || hostID == "0" {
			return MappingResult{Supported: false, Reason: "host_id 错误"}
		}
		return MappingResult{Supported: true}
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
		if memory := memoryMBToV1GB(payload, "memory"); memory != "" {
			values.Set("memory", memory)
		}
		if totalDisk := sumFields(payload, "sys_disk_size", "data_disk_size"); totalDisk != "" {
			values.Set("hard_disks", totalDisk)
		} else {
			setIf("sys_disk_size", "hard_disks")
		}
		setIf("data_disk_size", "data_disk_size")
		if v := firstNonZero(payload, "net_out", "net_in"); v != "" {
			values.Set("bandwidth", v)
		}
		setIf("expire_time", "expire_time")
		setIf("host_name", "host_name")
		setIf("sys_pwd", "sys_pwd")
		setIf("snapshot", "snapshot")
		setIf("backups", "backups")
		setIf("max_reinstall_num", "max_reinstall_num")
		setIf("flow_limit", "traffic")
		setIf("port_num", "port_num")
		setIf("domain_num", "domain_num")
		if shouldMapPublicIPCount(payload) {
			setIf("ip_num", "ipnum")
		}
		return queryReq(http.MethodPost, "create_host")
	case "updateHost":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("cpu", "cpu")
		if memory := memoryMBToV1GB(payload, "memory"); memory != "" {
			values.Set("memory", memory)
		}
		if totalDisk := sumFields(payload, "sys_disk_size", "data_disk_size"); totalDisk != "" {
			values.Set("hard_disks", totalDisk)
		} else {
			setIf("sys_disk_size", "hard_disks")
		}
		setIf("data_disk_size", "data_disk_size")
		if v := firstNonZero(payload, "net_out", "net_in"); v != "" {
			values.Set("bandwidth", v)
		}
		setIf("backups", "backups")
		setIf("snapshot", "snapshot")
		setIf("port_num", "port_num")
		if shouldMapPublicIPCount(payload) {
			setIf("ip_num", "ip_num")
		}
		return queryReq(http.MethodPost, "elastic_update")
	case "removeHost":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		return formReq(http.MethodPost, "delete")
	case "info":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		return formReq(http.MethodPost, "hostinfo")
	case "renew":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("nextduedate", "nextduedate")
		return formReq(http.MethodPost, "renew")
	case "power":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
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
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		return formReq(http.MethodPost, "monitor")
	case "updateOSPassword":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("password", "password")
		return formReq(http.MethodPost, "reset_password")
	case "osList":
		setIf("line_id", "line_id")
		return queryReq(http.MethodGet, "mirror_image")
	case "installOS":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("template", "template_id")
		setIf("password", "password")
		return formReq(http.MethodPost, "reset_os")
	case "snapshot":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		return formReq(http.MethodPost, "snapshot_list")
	case "createSnapshot":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		return formReq(http.MethodPost, "snapshot_add")
	case "removeSnapshot":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "snapshot_del")
	case "restoreSnapshot":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "snapshot_restore")
	case "backup":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		return formReq(http.MethodPost, "backups_list")
	case "createBackup":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		return formReq(http.MethodPost, "backups_add")
	case "removeBackup":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "backups_del")
	case "restoreBackupHost":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "backups_restore")
	case "firewallList":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("page", "page")
		setIf("direction", "direction")
		setIf("method", "method")
		setIf("protocol", "protocol")
		return queryReq(http.MethodGet, "security_acl_list")
	case "addFirewall":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
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
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "security_acl_del")
	case "portList":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		return queryReq(http.MethodGet, "nat_acl_list")
	case "addPort":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("dport", "dport")
		setIf("sport", "sport")
		setIf("name", "name")
		return formReq(http.MethodPost, "add_port_host")
	case "removePort":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("id", "id")
		return formReq(http.MethodPost, "remove_port_host")
	case "findport":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		setIf("keywords", "keywords")
		return queryReq(http.MethodGet, "findport")
	case "vnc":
		if invalid := requireHostID(); !invalid.Supported {
			return invalid
		}
		setHostID()
		return formReq(http.MethodPost, "vnc_view")
	case "panel":
		setAll()
		return formReq(http.MethodPost, "panel")
	case "test":
		setAll()
		return formReq(http.MethodPost, "test")
	case "thumbnail", "historyNetwork", "historyCpu", "synctime", "updatePanelPassword",
		"isoList", "mountISO", "bios", "addIP", "removeIP", "domainList", "addDomain", "removeDomain":
		return MappingResult{Supported: false, Reason: "old upstream has no public api implementation for this endpoint"}
	default:
		return MappingResult{Supported: false, Reason: "endpoint behavior not audited yet"}
	}
}

func first(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if v := stringOf(payload, key); v != "" {
			return v
		}
	}
	return ""
}

func firstNonZero(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		v := stringOf(payload, key)
		if v == "" || v == "0" {
			continue
		}
		return v
	}
	return first(payload, keys...)
}

func sumFields(payload map[string]any, keys ...string) string {
	var total int64
	var hasValue bool
	for _, key := range keys {
		raw := stringOf(payload, key)
		if raw == "" {
			continue
		}
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			continue
		}
		total += val
		hasValue = true
	}
	if !hasValue {
		return ""
	}
	return strconv.FormatInt(total, 10)
}

func shouldMapPublicIPCount(payload map[string]any) bool {
	isNAT := strings.TrimSpace(stringOf(payload, "is_nat"))
	return isNAT == "" || isNAT == "0" || strings.EqualFold(isNAT, "false")
}

func memoryMBToV1GB(payload map[string]any, key string) string {
	raw := stringOf(payload, key)
	if raw == "" {
		return ""
	}
	mb, err := strconv.ParseFloat(raw, 64)
	if err != nil || mb < 1024 {
		return raw
	}
	gb := int64(math.Round(mb / 1024))
	if gb < 1 {
		gb = 1
	}
	return strconv.FormatInt(gb, 10)
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
