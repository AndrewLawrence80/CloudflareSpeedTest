package model

import "gorm.io/gorm"

type DownloadSummary struct {
	gorm.Model
	IP        string  `gorm:"column:ip" json:"ip"`
	Bandwidth float64 `gorm:"column:bandwidth" json:"bandwidth"`
}
