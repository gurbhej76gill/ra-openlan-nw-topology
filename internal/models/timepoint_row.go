package models

type TimepointRow struct {
	ID         string
	BoardID    string
	Timestamp  int64  // epoch seconds (UTC)
	SSIDData   string // JSON array (faces)
	DeviceInfo string // JSON object
	Serial     string
}
