package models

import "time"

type timepointRow struct { // unexported: internal usage only
	ID           string         `json:"id"`
	BoardID      string         `json:"boardId"`
	Timestamp    time.Time      `json:"timestamp"`
	SerialNumber string         `json:"serialNumber"`
	SSIDData     map[string]any `json:"ssidData"`   // phase-1 input; flexible
	DeviceInfo   map[string]any `json:"deviceInfo"` // optional fields for enrichment
}

// RowLike is the minimal view the service needs.
type RowLike interface {
	SSID() map[string]any
}

func (r row) SSID() map[string]any { return r.ssidData }
