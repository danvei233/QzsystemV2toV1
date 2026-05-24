# v2 -> v1 接口逐条核对

基准来源：

- v2 文档：`默认模块.openapi.json`
- 旧接口源码：`D:/项目/golang/xiaohei/qz-override/app/api/controller/Cloud.php`
- 当前网关实现：`internal/app/gateway/mapper.go`、`internal/app/gateway/service.go`

状态定义：

- `已对齐`：已有明确映射，输入输出语义基本可落地
- `部分对齐`：有映射，但输入或输出仍有语义缺口
- `未对齐`：当前没有可靠实现，或旧接口源码里没有可直接对应的方法

## 核对结果

| v2 接口 | 旧接口源码对应 | 当前网关 | 状态 | 关键问题 |
| --- | --- | --- | --- | --- |
| `/api/v1/openHost` | `create_host()` / `openhost()` | `create_host` | 部分对齐 | 旧接口按 `os_name` 查模板，不是简单字段改名；已补线路镜像预解析，但输出还没按 v2 结构重组；`cpu_limit/ip_num/flow_limit/buy_time/is_nat/max_reinstall_num` 等未完整映射 |
| `/api/v1/updateHost` | `elastic_update()` / `update()` | `elastic_update` | 部分对齐 | 只映射了 `cpu/memory/sys_disk_size/data_disk_size/net_out/net_in/port_num`；`snapshot/backups/flow_limit/domain_num` 没处理；输出未按 v2 文档重组 |
| `/api/v1/removeHost` | `delete()` | `delete` | 已对齐 | 基本动作没问题，输出仍是旧接口原样 |
| `/api/v1/info` | `hostinfo($host_id)` | `hostinfo` | 部分对齐 | 输入已映射 `hostid -> host_id/hostid`；输出目前还是旧接口透传，未严格整理成 v2 文档要求字段 |
| `/api/v1/renew` | `renew($host_id,$nextduedate)` | `renew` | 已对齐 | 输入结构可用，输出仍透传 |
| `/api/v1/power` | `start()` / `reboot()` / `shutdown()` | `start/reboot/shutdown` | 部分对齐 | `state=2`、`5` 明确；`3软关机`、`4断电` 目前都压成 `shutdown` |
| `/api/v1/monitor` | `monitor($host_id)` | `monitor` | 部分对齐 | 能调通旧接口，但输出没有按 v2 文档字段标准化 |
| `/api/v1/thumbnail` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码里没找到等价实现 |
| `/api/v1/historyNetwork` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码里没找到等价实现 |
| `/api/v1/historyCpu` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码里没找到等价实现 |
| `/api/v1/synctime` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码里没找到等价实现 |
| `/api/v1/updateOSPassword` | `reset_password($host_id)` | `reset_password` | 已对齐 | 输入可用，输出仍透传 |
| `/api/v1/updatePanelPassword` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码里没有独立 `panel_password` 修改方法 |
| `/api/v1/osList` | `mirror_image()` | `mirror_image` | 部分对齐 | 旧接口按 `line_id` 返回镜像列表，这点可用；但 v2 文档里也出现过按 `hostid` 查询的语义，当前未兼容 |
| `/api/v1/installOS` | `reset_os($host_id,$template_id,$password)` | `reset_os` | 部分对齐 | 旧接口内部支持 `id 或 os_name`，当前传参可工作，但输出未按 v2 文档重塑 |
| `/api/v1/isoList` | `cdrom($host_id=0)` | 同名兜底 | 未对齐 | 旧源码只有 `cdrom()` 返回空数组，不是 v2 语义的 iso 文件列表 |
| `/api/v1/mountISO` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码没有挂载/卸载 ISO 逻辑 |
| `/api/v1/bios` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码没有 BIOS 修改逻辑 |
| `/api/v1/addIP` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码没有增加附加 IP 接口 |
| `/api/v1/removeIP` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码没有删除附加 IP 接口 |
| `/api/v1/snapshot` | `snapshot_list($host_id)` | `snapshot_list` | 已对齐 | 输入动作可用，输出仍透传 |
| `/api/v1/createSnapshot` | `snapshot_add($host_id)` | `snapshot_add` | 已对齐 | 输入动作可用，输出仍透传 |
| `/api/v1/removeSnapshot` | `snapshot_del($host_id,$id)` | `snapshot_del` | 已对齐 | 输入动作可用，输出仍透传 |
| `/api/v1/restoreSnapshot` | `snapshot_restore($host_id,$id)` | `snapshot_restore` | 已对齐 | 输入动作可用，输出仍透传 |
| `/api/v1/backup` | `backups_list($host_id)` | `backups_list` | 已对齐 | 输入动作可用，输出仍透传 |
| `/api/v1/createBackup` | `backups_add($host_id)` | `backups_add` | 已对齐 | 输入动作可用，输出仍透传 |
| `/api/v1/removeBackup` | `backups_del($host_id,$id)` | `backups_del` | 已对齐 | 输入动作可用，输出仍透传 |
| `/api/v1/restoreBackupHost` | `backups_restore($host_id,$id)` | `backups_restore` | 已对齐 | 输入动作可用，输出仍透传 |
| `/api/v1/firewallList` | `security_acl_list($host_id)` | `security_acl_list` | 部分对齐 | 旧接口不吃 `page/direction/method/protocol` 过滤语义，当前只是附带透传 |
| `/api/v1/addFirewall` | `security_acl_add()` | `security_acl_add` | 部分对齐 | 旧接口支持较多参数，但当前只做基础透传，未验证优先级/默认值行为 |
| `/api/v1/removeFirewall` | `security_acl_del($host_id,$id)` | `security_acl_del` | 已对齐 | 基本动作可用 |
| `/api/v1/portList` | `nat_acl_list($host_id)` | `nat_acl_list` | 已对齐 | 基本动作可用 |
| `/api/v1/addPort` | `add_port_host()` | `add_port_host` | 已对齐 | 基本动作可用 |
| `/api/v1/removePort` | `remove_port_host()` | `remove_port_host` | 已对齐 | 基本动作可用 |
| `/api/v1/findport` | `findport()` | `findport` | 已对齐 | 基本动作可用 |
| `/api/v1/domainList` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码没有域名列表接口 |
| `/api/v1/addDomain` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码没有域名添加接口 |
| `/api/v1/removeDomain` | 无直接对应 | 同名兜底 | 未对齐 | 旧源码没有域名删除接口 |
| `/api/v1/vnc` | `vnc_view($host_id)` | `vnc_view` | 已对齐 | 302 已包装成 JSON URL |
| `/api/v1/panel` | `panel()` | `panel` | 已对齐 | 302 已包装成 JSON URL |
| `/api/v1/test` | `test()` | `test` | 已对齐 | 调试用途 |

## 当前最危险的接口

### 1. `openHost`

不是简单 `os_name -> os` 就结束。

旧源码真实行为：

- 先按 `line_id` / `nodes_id` 选线路和节点
- 再用 `ServersImageConfig.where(os_name = post['os'])` 查镜像模板
- 找不到就直接报 `image template not found`

也就是说，创建前必须考虑：

- 镜像名是否是旧面板数据库里的真实 `os_name`
- 镜像是否在该 `line_id` 的镜像集合里可用
- `nodes_id` 是否影响可用模板

### 2. `info`

旧接口能返回大量字段，但当前网关只是透传旧 JSON。
v2 文档要求的数据层级和字段语义更稳定，后面需要做专门 DTO 重组。

### 3. `power`

当前只把：

- `2 -> start`
- `5 -> reboot`

做了明确映射。

但 v2 文档还有：

- `3 -> 软关机`
- `4 -> 断电`

这两个现在没有区分。

## 已经确认的事实

### `openHost` 最近真实请求

最近 `v2.sermc` 请求中，网关实际转发为：

- `create_host?...&line_id=7&nodes_id=8&os=ubuntu20.04`
- 返回：`{"code":-1,"msg":"image template not found"}`

所以当前没有“最近一次成功开通”的真实日志样本，最近样本全是失败。

### 旧源码对镜像的真实要求

- `create_host()`：要求 `post['os']` 是旧库 `ServersImageConfig.os_name`
- `reset_os()`：支持 `id` 或 `os_name`
- `mirror_image()`：按 `line_id` 给出该线路可用镜像清单

## 后续必须补的事情

1. `openHost` 输出 DTO 重组
2. `info` 输出 DTO 重组
3. `power` 区分 `软关机/断电`
4. `updateHost` 补齐未映射字段
5. 对 `thumbnail/historyNetwork/historyCpu/synctime/updatePanelPassword/isoList/mountISO/bios/addIP/removeIP/domainList/addDomain/removeDomain` 明确：
   要么实现
   要么在网关层显式返回 `not implemented`，不能继续靠同名兜底赌运气

