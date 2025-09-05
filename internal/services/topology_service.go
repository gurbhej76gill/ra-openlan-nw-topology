package services

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/router-architects/network-topology/internal/config"
	"github.com/router-architects/network-topology/internal/models"
	"github.com/router-architects/network-topology/internal/repositories"
	"github.com/router-architects/network-topology/internal/utils"
)

type TopologyService interface {
	Build(ctx context.Context, groupID string, at time.Time) (models.TopologyResponse, error)
}

type topologyService struct {
	repo repositories.TopologyRepository
	cfg  config.Config
}

func NewTopologyService(r repositories.TopologyRepository, cfg config.Config) TopologyService {
	return &topologyService{repo: r, cfg: cfg}
}

func (s *topologyService) Build(ctx context.Context, groupID string, at time.Time) (models.TopologyResponse, error) {
	windowStart, windowEnd := utils.RollingWindow(at, s.cfg.TopologyWindow, s.cfg.DriftAllowance)

	logrus.WithFields(logrus.Fields{
		"groupId": groupID, "start": windowStart, "end": windowEnd,
	}).Debug("fetching latest rows in window")

	rows, err := s.repo.FetchLatestRows(ctx, groupID, windowStart, windowEnd)
	if err != nil {
		return models.TopologyResponse{}, fmt.Errorf("repo.FetchLatestRows: %w", err)
	}

	type faceKey struct {
		serial string
		bssid  string
	}

	// Intermediate maps guarded by RW locks for safe concurrent fill.
	facesBySerial := make(map[string][]models.Face)
	clientsByStation := make(map[string][]struct {
		serial    string
		faceBSSID string
		ts        time.Time
		rssi      int
	})
	knownMeshBSSIDs := make(map[string]string) // bssid -> serial
	muFaces := sync.RWMutex{}
	muClients := sync.RWMutex{}
	muMesh := sync.RWMutex{}

	wg := sync.WaitGroup{}
	wg.Add(len(rows))

	for i := range rows {
		rr := rows[i]
		go func() {
			defer wg.Done()
			parsedFaces := parseSSIDData(rr, rr.timestamp)

			// Fill facesBySerial
			muFaces.Lock()
			facesBySerial[rr.serialNumber] = append(facesBySerial[rr.serialNumber], parsedFaces...)
			muFaces.Unlock()

			// Record mesh bssids
			muMesh.Lock()
			for _, f := range parsedFaces {
				if f.Mode == "mesh" && f.BSSID != "" {
					knownMeshBSSIDs[f.BSSID] = rr.serialNumber
				}
			}
			muMesh.Unlock()

			// Fill clientsByStation
			muClients.Lock()
			for _, f := range parsedFaces {
				for _, c := range f.Clients {
					clientsByStation[c.Station] = append(clientsByStation[c.Station], struct {
						serial    string
						faceBSSID string
						ts        time.Time
						rssi      int
					}{rr.serialNumber, f.BSSID, rr.timestamp, c.RSSI})
				}
			}
			muClients.Unlock()
		}()
	}
	wg.Wait()

	// Deduplicate clients: choose latest ts, then highest rssi.
	finalClientsByBSSID := make(map[string]map[string]struct{})
	for station, assoc := range clientsByStation {
		sort.Slice(assoc, func(i, j int) bool {
			if assoc[i].ts.Equal(assoc[j].ts) {
				return assoc[i].rssi > assoc[j].rssi
			}
			return assoc[i].ts.After(assoc[j].ts)
		})
		chosen := assoc[0]
		if _, ok := finalClientsByBSSID[chosen.faceBSSID]; !ok {
			finalClientsByBSSID[chosen.faceBSSID] = make(map[string]struct{})
		}
		finalClientsByBSSID[chosen.faceBSSID][station] = struct{}{}
	}

	// Filter face clients to only chosen stations
	for serial, faces := range facesBySerial {
		for i := range faces {
			f := &faces[i]
			if set, ok := finalClientsByBSSID[f.BSSID]; ok {
				filtered := make([]models.Client, 0, len(f.Clients))
				for _, c := range f.Clients {
					if _, keep := set[c.Station]; keep {
						filtered = append(filtered, c)
					}
				}
				f.Clients = filtered
			} else {
				f.Clients = nil
			}
		}
		facesBySerial[serial] = faces
	}

	// Construct mesh edges
	meshEdges := make([]models.MeshEdge, 0, 16)
	for serial, faces := range facesBySerial {
		for _, f := range faces {
			if f.Mode != "mesh" {
				continue
			}
			for _, c := range f.Clients {
				if peerSerial, ok := knownMeshBSSIDs[c.Station]; ok {
					meshEdges = append(meshEdges, models.MeshEdge{
						From: serial, To: peerSerial,
						SSID: f.SSID, Band: f.Band, Channel: f.Channel,
					})
				}
			}
		}
	}

	// Build nodes
	nodes := make([]models.TopologyNode, 0, len(facesBySerial))
	for serial, faces := range facesBySerial {
		var aps, mesh []models.Face
		for _, f := range faces {
			if f.Mode == "mesh" {
				mesh = append(mesh, f)
			} else {
				aps = append(aps, f)
			}
		}
		nodes = append(nodes, models.TopologyNode{
			Serial: serial,
			APs:    aps,
			Mesh:   mesh,
		})
	}

	mode := "latest"
	if !at.IsZero() && at.Before(time.Now().UTC().Add(-1*time.Minute)) {
		mode = "historical"
	}

	var resp models.TopologyResponse
	resp.GroupID = groupID
	resp.Timestamp = at.UTC().Format(time.RFC3339)
	resp.Meta = struct {
		Mode                  string    `json:"mode"`
		WindowStart           time.Time `json:"window_start"`
		WindowEnd             time.Time `json:"window_end"`
		DriftAllowanceSeconds int       `json:"drift_allowance_seconds"`
		ServedFrom            string    `json:"served_from"`
	}{
		Mode:                  mode,
		WindowStart:           windowStart,
		WindowEnd:             windowEnd,
		DriftAllowanceSeconds: int(s.cfg.DriftAllowance / time.Second),
		ServedFrom:            "db",
	}
	resp.Nodes = nodes
	resp.Edges.Wired = []any{}
	resp.Edges.Mesh = meshEdges
	resp.External = []any{}

	return resp, nil
}

// parseSSIDData converts the flexible ssid_data map to faces and clients.
// We keep it defensive: tolerate missing/variant fields.
func parseSSIDData(rr repositories.RowLike, ts time.Time) []models.Face {
	// Expecting structure similar to doc:
	// faces: [] with bssid, ssid, band, channel, mode, clients: []
	// Since repositories.row is unexported, expose a minimal RowLike interface to decouple.
	type clientIn struct {
		Station  string `json:"station"`
		RSSI     int    `json:"rssi"`
		Conn     int    `json:"connected"`
		Inactive int    `json:"inactive"`
		RxBR     int    `json:"rx_rate_bitrate"`
		TxBR     int    `json:"tx_rate_bitrate"`
		RxCW     int    `json:"rx_rate_chwidth"`
	}
	var faces []models.Face

	raw := rr.SSID()
	// Try common shapes:
	// 1) {"faces":[{...}]}
	// 2) {"aps":[{...}], "mesh":[{...}]}
	// 3) {"interfaces":[{...}]}
	// 4) Flat array: {"ssid_data":[{...}]}
	// We will attempt to detect arrays in known keys; else if ssid_data itself is an array.

	extractFaces := func(arr any, modeHint string) {
		as, ok := arr.([]any)
		if !ok {
			return
		}
		for _, it := range as {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			f := models.Face{
				BSSID:     str(m["bssid"]),
				SSID:      str(m["ssid"]),
				Band:      str(m["band"]),
				Channel:   intval(m["channel"]),
				Mode:      firstNonEmpty(str(m["mode"]), modeHint),
				Timestamp: ts,
			}

			if cs, ok := m["clients"]; ok {
				if arrc, ok := cs.([]any); ok {
					for _, c := range arrc {
						cm, ok := c.(map[string]any)
						if !ok {
							continue
						}
						f.Clients = append(f.Clients, models.Client{
							Station:       str(cm["station"]),
							RSSI:          intval(cm["rssi"]),
							Connected:     intval(cm["connected"]),
							Inactive:      intval(cm["inactive"]),
							RxRateBitrate: intval(cm["rx_rate_bitrate"]),
							TxRateBitrate: intval(cm["tx_rate_bitrate"]),
							RxRateChWidth: intval(cm["rx_rate_chwidth"]),
						})
					}
				}
			}
			faces = append(faces, f)
		}
	}

	if v, ok := raw["faces"]; ok {
		extractFaces(v, "")
	}
	if v, ok := raw["aps"]; ok {
		extractFaces(v, "ap")
	}
	if v, ok := raw["mesh"]; ok {
		extractFaces(v, "mesh")
	}
	if v, ok := raw["interfaces"]; ok && len(faces) == 0 {
		extractFaces(v, "")
	}
	// if nothing yet, try if ssid_data itself is an array
	if len(faces) == 0 {
		extractFaces(raw["ssid_data"], "")
	}

	return faces
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
func intval(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		return 0
	default:
		return 0
	}
}
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
