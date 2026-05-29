package repository

import (
	"context"
	"time"

	"xiaoheiproxy/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HostV2MetadataRow struct {
	HostID            uint `gorm:"primaryKey"`
	LineID            int
	NodesID           int
	HostName          string
	OSName            string
	CPU               int
	CPULimit          int
	MemoryMB          int
	SysDiskSizeGB     int
	DataDiskSizeGB    int
	SysDiskIOPS       int
	DataDiskIOPS      int
	NetOutMbps        int
	NetInMbps         int
	FlowLimitGB       int
	IPNum             int
	IsNAT             int
	PortNum           int
	DomainNum         int
	SnapshotNum       int
	BackupNum         int
	MaxReinstallNum   int
	BuyTime           string
	ExpireTime        string
	LastOpenRequest   string `gorm:"type:text"`
	LastUpdateRequest string `gorm:"type:text"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (HostV2MetadataRow) TableName() string { return "host_v2_metadata" }

type SQLiteHostV2MetadataRepository struct {
	db *gorm.DB
}

func NewHostV2MetadataRepository(db *gorm.DB) *SQLiteHostV2MetadataRepository {
	return &SQLiteHostV2MetadataRepository{db: db}
}

func (r *SQLiteHostV2MetadataRepository) Upsert(ctx context.Context, metadata *domain.HostV2Metadata) error {
	row := toHostMetadataRow(*metadata)
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now()
	}
	row.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "host_id"}},
		UpdateAll: true,
	}).Create(&row).Error
}

func (r *SQLiteHostV2MetadataRepository) Get(ctx context.Context, hostID uint) (*domain.HostV2Metadata, error) {
	var row HostV2MetadataRow
	if err := r.db.WithContext(ctx).First(&row, "host_id = ?", hostID).Error; err != nil {
		return nil, err
	}
	metadata := fromHostMetadataRow(row)
	return &metadata, nil
}

func (r *SQLiteHostV2MetadataRepository) Delete(ctx context.Context, hostID uint) error {
	return r.db.WithContext(ctx).Delete(&HostV2MetadataRow{}, "host_id = ?", hostID).Error
}

func toHostMetadataRow(metadata domain.HostV2Metadata) HostV2MetadataRow {
	return HostV2MetadataRow{
		HostID:            metadata.HostID,
		LineID:            metadata.LineID,
		NodesID:           metadata.NodesID,
		HostName:          metadata.HostName,
		OSName:            metadata.OSName,
		CPU:               metadata.CPU,
		CPULimit:          metadata.CPULimit,
		MemoryMB:          metadata.MemoryMB,
		SysDiskSizeGB:     metadata.SysDiskSizeGB,
		DataDiskSizeGB:    metadata.DataDiskSizeGB,
		SysDiskIOPS:       metadata.SysDiskIOPS,
		DataDiskIOPS:      metadata.DataDiskIOPS,
		NetOutMbps:        metadata.NetOutMbps,
		NetInMbps:         metadata.NetInMbps,
		FlowLimitGB:       metadata.FlowLimitGB,
		IPNum:             metadata.IPNum,
		IsNAT:             metadata.IsNAT,
		PortNum:           metadata.PortNum,
		DomainNum:         metadata.DomainNum,
		SnapshotNum:       metadata.SnapshotNum,
		BackupNum:         metadata.BackupNum,
		MaxReinstallNum:   metadata.MaxReinstallNum,
		BuyTime:           metadata.BuyTime,
		ExpireTime:        metadata.ExpireTime,
		LastOpenRequest:   metadata.LastOpenRequest,
		LastUpdateRequest: metadata.LastUpdateRequest,
		CreatedAt:         metadata.CreatedAt,
		UpdatedAt:         metadata.UpdatedAt,
	}
}

func fromHostMetadataRow(row HostV2MetadataRow) domain.HostV2Metadata {
	return domain.HostV2Metadata{
		HostID:            row.HostID,
		LineID:            row.LineID,
		NodesID:           row.NodesID,
		HostName:          row.HostName,
		OSName:            row.OSName,
		CPU:               row.CPU,
		CPULimit:          row.CPULimit,
		MemoryMB:          row.MemoryMB,
		SysDiskSizeGB:     row.SysDiskSizeGB,
		DataDiskSizeGB:    row.DataDiskSizeGB,
		SysDiskIOPS:       row.SysDiskIOPS,
		DataDiskIOPS:      row.DataDiskIOPS,
		NetOutMbps:        row.NetOutMbps,
		NetInMbps:         row.NetInMbps,
		FlowLimitGB:       row.FlowLimitGB,
		IPNum:             row.IPNum,
		IsNAT:             row.IsNAT,
		PortNum:           row.PortNum,
		DomainNum:         row.DomainNum,
		SnapshotNum:       row.SnapshotNum,
		BackupNum:         row.BackupNum,
		MaxReinstallNum:   row.MaxReinstallNum,
		BuyTime:           row.BuyTime,
		ExpireTime:        row.ExpireTime,
		LastOpenRequest:   row.LastOpenRequest,
		LastUpdateRequest: row.LastUpdateRequest,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}
