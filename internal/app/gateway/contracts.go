package gateway

import "strings"

type DataShape string

const (
	DataShapeNone       DataShape = "none"
	DataShapeObject     DataShape = "object"
	DataShapeArray      DataShape = "array"
	DataShapeURL        DataShape = "object:url"
	DataShapeDataExtend DataShape = "object:data+extend"
)

type ConflictPolicy string

const (
	ConflictPolicyExample  ConflictPolicy = "example"
	ConflictPolicyFunction ConflictPolicy = "function"
)

type EndpointContract struct {
	Path           string
	SuccessCode    int
	SuccessMsg     string
	HasData        bool
	Shape          DataShape
	ConflictPolicy ConflictPolicy
	HasExample     bool
}

var endpointContracts = map[string]EndpointContract{
	"/api/v1/openHost":            {Path: "/api/v1/openHost", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeObject, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/updateHost":          {Path: "/api/v1/updateHost", SuccessCode: 0, SuccessMsg: "success", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/removeHost":          {Path: "/api/v1/removeHost", SuccessCode: 0, SuccessMsg: "success", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/info":                {Path: "/api/v1/info", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeObject, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/renew":               {Path: "/api/v1/renew", SuccessCode: 0, SuccessMsg: "success", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/power":               {Path: "/api/v1/power", SuccessCode: 200, SuccessMsg: "启动命令执行成功", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/monitor":             {Path: "/api/v1/monitor", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeObject, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/thumbnail":           {Path: "/api/v1/thumbnail", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeObject, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/historyNetwork":      {Path: "/api/v1/historyNetwork", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/historyCpu":          {Path: "/api/v1/historyCpu", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/synctime":            {Path: "/api/v1/synctime", SuccessCode: 200, SuccessMsg: "success", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/updateOSPassword":    {Path: "/api/v1/updateOSPassword", SuccessCode: 200, SuccessMsg: "success", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/updatePanelPassword": {Path: "/api/v1/updatePanelPassword", SuccessCode: 200, SuccessMsg: "success", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/osList":              {Path: "/api/v1/osList", SuccessCode: 0, SuccessMsg: "success", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyFunction, HasExample: true},
	"/api/v1/installOS":           {Path: "/api/v1/installOS", SuccessCode: 200, SuccessMsg: "执行重装系统命令成功", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/isoList":             {Path: "/api/v1/isoList", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeObject, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/mountISO":            {Path: "/api/v1/mountISO", SuccessCode: 200, SuccessMsg: "success", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/bios":                {Path: "/api/v1/bios", SuccessCode: 200, SuccessMsg: "success", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/addIP":               {Path: "/api/v1/addIP", SuccessCode: 200, SuccessMsg: "success", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/removeIP":            {Path: "/api/v1/removeIP", SuccessCode: 200, SuccessMsg: "success", HasData: false, Shape: DataShapeNone, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/snapshot":            {Path: "/api/v1/snapshot", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeDataExtend, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/createSnapshot":      {Path: "/api/v1/createSnapshot", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeDataExtend, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/removeSnapshot":      {Path: "/api/v1/removeSnapshot", SuccessCode: 200, SuccessMsg: "执行删除快照命令成功", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/restoreSnapshot":     {Path: "/api/v1/restoreSnapshot", SuccessCode: 200, SuccessMsg: "执行恢复快照命令成功", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/backup":              {Path: "/api/v1/backup", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeDataExtend, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/createBackup":        {Path: "/api/v1/createBackup", SuccessCode: 200, SuccessMsg: "执行创建备份命令成功", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/removeBackup":        {Path: "/api/v1/removeBackup", SuccessCode: 200, SuccessMsg: "执行删除备份命令成功", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/restoreBackupHost":   {Path: "/api/v1/restoreBackupHost", SuccessCode: 200, SuccessMsg: "执行恢复备份命令成功", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/firewallList":        {Path: "/api/v1/firewallList", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeDataExtend, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/addFirewall":         {Path: "/api/v1/addFirewall", SuccessCode: 200, SuccessMsg: "添加策略成功", HasData: true, Shape: DataShapeDataExtend, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/removeFirewall":      {Path: "/api/v1/removeFirewall", SuccessCode: 200, SuccessMsg: "删除策略成功", HasData: true, Shape: DataShapeDataExtend, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/portList":            {Path: "/api/v1/portList", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/addPort":             {Path: "/api/v1/addPort", SuccessCode: 200, SuccessMsg: "添加端口成功", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/removePort":          {Path: "/api/v1/removePort", SuccessCode: 200, SuccessMsg: "删除端口成功", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/findport":            {Path: "/api/v1/findport", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/domainList":          {Path: "/api/v1/domainList", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/addDomain":           {Path: "/api/v1/addDomain", SuccessCode: 200, SuccessMsg: "添加域名成功", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/removeDomain":        {Path: "/api/v1/removeDomain", SuccessCode: 200, SuccessMsg: "删除域名成功", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/vnc":                 {Path: "/api/v1/vnc", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeURL, ConflictPolicy: ConflictPolicyExample, HasExample: true},
	"/api/v1/panel":               {Path: "/api/v1/panel", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeURL, ConflictPolicy: ConflictPolicyFunction, HasExample: false},
	"/api/v1/test":                {Path: "/api/v1/test", SuccessCode: 200, SuccessMsg: "success", HasData: true, Shape: DataShapeArray, ConflictPolicy: ConflictPolicyExample, HasExample: true},
}

func contractFor(path string) EndpointContract {
	if contract, ok := endpointContracts[path]; ok {
		return contract
	}
	return EndpointContract{
		Path:           path,
		SuccessCode:    200,
		SuccessMsg:     "success",
		HasData:        false,
		Shape:          DataShapeNone,
		ConflictPolicy: ConflictPolicyFunction,
		HasExample:     false,
	}
}

func isKnownEndpoint(path string) bool {
	_, ok := endpointContracts[path]
	return ok
}

func emptyDataForShape(shape DataShape) any {
	switch shape {
	case DataShapeArray:
		return []any{}
	case DataShapeURL:
		return map[string]any{"url": ""}
	case DataShapeDataExtend:
		return map[string]any{
			"data":   []any{},
			"extend": map[string]any{"count": 0},
		}
	case DataShapeObject:
		return map[string]any{}
	default:
		return nil
	}
}

func responseLooksLikeHTML(body []byte) bool {
	text := strings.TrimSpace(strings.ToLower(string(body)))
	return strings.HasPrefix(text, "<!doctype") || strings.HasPrefix(text, "<html") || strings.HasPrefix(text, "<head") || strings.HasPrefix(text, "<body")
}
