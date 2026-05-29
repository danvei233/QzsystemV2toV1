package domain

import "time"

type HostV2Metadata struct {
	HostID            uint      `json:"host_id"`
	LineID            int       `json:"line_id"`
	NodesID           int       `json:"nodes_id"`
	HostName          string    `json:"host_name"`
	OSName            string    `json:"os_name"`
	CPU               int       `json:"cpu"`
	CPULimit          int       `json:"cpu_limit"`
	MemoryMB          int       `json:"memory_mb"`
	SysDiskSizeGB     int       `json:"sys_disk_size_gb"`
	DataDiskSizeGB    int       `json:"data_disk_size_gb"`
	SysDiskIOPS       int       `json:"sys_disk_iops"`
	DataDiskIOPS      int       `json:"data_disk_iops"`
	NetOutMbps        int       `json:"net_out_mbps"`
	NetInMbps         int       `json:"net_in_mbps"`
	FlowLimitGB       int       `json:"flow_limit_gb"`
	IPNum             int       `json:"ip_num"`
	IsNAT             int       `json:"is_nat"`
	PortNum           int       `json:"port_num"`
	DomainNum         int       `json:"domain_num"`
	SnapshotNum       int       `json:"snapshot_num"`
	BackupNum         int       `json:"backup_num"`
	MaxReinstallNum   int       `json:"max_reinstall_num"`
	BuyTime           string    `json:"buy_time"`
	ExpireTime        string    `json:"expire_time"`
	LastOpenRequest   string    `json:"last_open_request"`
	LastUpdateRequest string    `json:"last_update_request"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
