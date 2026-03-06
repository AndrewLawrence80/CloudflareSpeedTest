package model

import "gorm.io/gorm"

type ICMPingSummary struct {
	gorm.Model
	IP         string  `gorm:"column:ip" json:"ip"`
	MinRTT     float64 `gorm:"column:min_rtt" json:"min_rtt"`
	AvgRTT     float64 `gorm:"column:avg_rtt" json:"avg_rtt"`
	MaxRTT     float64 `gorm:"column:max_rtt" json:"max_rtt"`
	PacketLoss float64 `gorm:"column:packet_loss" json:"packet_loss"`
}
