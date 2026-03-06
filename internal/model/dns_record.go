package model

import "gorm.io/gorm"

type DNSRecord struct {
	gorm.Model
	Domain       string   `gorm:"column:domain;uniqueIndex" json:"domain"` // use unique index to ensure upsert behavior
	IPv4         []string `gorm:"column:ipv4;type:text;serializer:json" json:"ipv4"`
	IPv6         []string `gorm:"column:ipv6;type:longtext;serializer:json" json:"ipv6"`
	Success      bool     `gorm:"column:success;index" json:"success"`
	IsCloudflare bool     `gorm:"column:is_cloudflare;index" json:"is_cloudflare"`
}

func (DNSRecord) TableName() string {
	return "dns_records"
}
