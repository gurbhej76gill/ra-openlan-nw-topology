package models

import "time"

// Public response DTOs can be exported; internal structs kept unexported to minimize API surface.

type Client struct {
	Station       string `json:"station"`
	RSSI          int    `json:"rssi"`
	Connected     int    `json:"connected"`
	Inactive      int    `json:"inactive"`
	RxRateBitrate int    `json:"rx_rate_bitrate"`
	TxRateBitrate int    `json:"tx_rate_bitrate"`
	RxRateChWidth int    `json:"rx_rate_chwidth"`
}

type Face struct {
	BSSID     string    `json:"bssid"`
	SSID      string    `json:"ssid"`
	Band      string    `json:"band"`
	Channel   int       `json:"channel"`
	Mode      string    `json:"mode"` // "ap" or "mesh"
	Clients   []Client  `json:"clients"`
	Timestamp time.Time `json:"timestamp"`
}

type TopologyNode struct {
	Serial string `json:"serial"`
	APs    []Face `json:"aps"`
	Mesh   []Face `json:"mesh"`
}

type MeshEdge struct {
	From    string `json:"from"`
	To      string `json:"to"`
	SSID    string `json:"ssid"`
	Band    string `json:"band"`
	Channel int    `json:"channel"`
}

type topologyMeta struct {
	Mode                  string    `json:"mode"` // latest|historical
	WindowStart           time.Time `json:"window_start"`
	WindowEnd             time.Time `json:"window_end"`
	DriftAllowanceSeconds int       `json:"drift_allowance_seconds"`
	ServedFrom            string    `json:"served_from"`
}

type TopologyResponse struct {
	GroupID   string       `json:"groupId"`
	Timestamp string       `json:"timestamp"`
	Meta      topologyMeta `json:"meta"`

	Nodes []TopologyNode `json:"nodes"`
	Edges struct {
		Wired []any      `json:"wired"`
		Mesh  []MeshEdge `json:"mesh"`
	} `json:"edges"`
	External []any `json:"external"`
}
