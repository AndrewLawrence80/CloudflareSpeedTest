package model

import "gorm.io/gorm"

type DNSRecord struct {
	gorm.Model
	Domain string   `gorm:"column:domain;index" json:"domain"`
	IPv4   []string `gorm:"column:ipv4;type:text;serializer:json" json:"ipv4"`
	IPv6   []string `gorm:"column:ipv6;type:longtext;serializer:json" json:"ipv6"`
}

func (DNSRecord) TableName() string {
	return "dns_records"
}
