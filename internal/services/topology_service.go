package services

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/router-architects/network-topology-service/internal/adapters/serviceclient"
	"github.com/router-architects/network-topology-service/internal/logger"
	"github.com/router-architects/network-topology-service/internal/models"
	"github.com/router-architects/network-topology-service/internal/repositories"
)

// Service interface
type TopologyService interface {
	BuildTopology(ctx context.Context, boardID string, params models.TimepointsQuery) (models.Topology, error)
}

type topologyService struct {
	repo   repositories.TopologyRepository
	client serviceclient.OpenAPIRequestClient
}

func NewTopologyService(repo repositories.TopologyRepository, client serviceclient.OpenAPIRequestClient) TopologyService {
	return &topologyService{repo: repo, client: client}
}

// ---------- Input JSON structures (ssid_data, device_info) ----------
type ssidFace struct {
	Associations []struct {
		Connected int    `json:"connected"`
		Inactive  int    `json:"inactive"`
		RSSI      int    `json:"rssi"`
		Station   string `json:"station"`
		RxRate    struct {
			Bitrate int `json:"bitrate"`
			Chwidth int `json:"chwidth"`
		} `json:"rx_rate"`
		TxRate struct {
			Bitrate int `json:"bitrate"`
			Chwidth int `json:"chwidth"`
		} `json:"tx_rate"`
	} `json:"associations"`
	Band    int    `json:"band"`
	BSSID   string `json:"bssid"`
	Channel int    `json:"channel"`
	Mode    string `json:"mode"` // "ap" | "mesh"
	SSID    string `json:"ssid"`
}

type deviceInfo struct {
	DeviceType   string `json:"deviceType"`
	SerialNumber string `json:"serialNumber"`
}

// ---------- Internal helpers ----------
type rowParsed struct {
	ts     int64
	serial string
	faces  []models.SSIDData
}

type faceKey struct {
	serial string
	bssid  string
	mode   string // ap | mesh
}

type faceOut struct {
	face    models.Face
	ts      int64
	clients []models.FaceClient
}

// main method
func (s *topologyService) BuildTopology(ctx context.Context, boardID string, params models.TimepointsQuery) (models.Topology, error) {
	log := logger.ForFunctionality("TOPOLOGY-SERVICE")
	if log != nil {
		log = log.WithFields(logger.Fields{"boardId": boardID})
	}

	end := time.Now().Unix()

	rows, err := s.client.GetTimepoints(ctx, models.TimepointRequest{
		BoardID:        boardID,
		FromDate:       StringPtr(params.FromDate),
		EndDate:        StringPtr(params.EndDate),
		MaxRecords:     IntrPtr(params.MaxRecords),
		StatsOnly:      false,
		PointsOnly:     true,
		PointStatsOnly: false,
	})
	if err != nil {
		if log != nil {
			log.WithError(err).Error("fetch timepoints failed")
		}
		return models.Topology{}, err
	}

	// 2) Parse rows newest->older, collect:
	//    - device serial
	//    - all faces (latest per serial+bssid+mode)
	//    - known BSSID set & owner mapping (bssid -> serial)
	parsed := make([]rowParsed, 0, len(rows))
	bssidOwner := map[string]string{} // bssid -> serial
	knownBSSID := map[string]struct{}{}

	for _, r := range rows {

		serial := strings.TrimSpace(r.Serial)
		if sn := strings.TrimSpace(r.DeviceInfo.SerialNumber); sn != "" {
			serial = sn
		}

		serial = strings.TrimSpace(serial)

		faces := r.SSIDData
		parsed = append(parsed, rowParsed{ts: r.Timestamp, serial: serial, faces: faces})

		for _, f := range faces {
			b := normMAC(f.BSSID)
			if b == "" {
				continue
			}
			knownBSSID[b] = struct{}{}
			// Only set owner once (newest sample wins, rows are DESC)
			if _, ok := bssidOwner[b]; !ok {
				bssidOwner[b] = serial
			}
		}
	}

	// 3) First pass: decide the "latest" face record per (serial,bssid,mode)
	//    Build faceOut map and per-device container
	faceMap := map[faceKey]*faceOut{}
	devMap := map[string]*models.Device{}

	for _, rp := range parsed {
		for _, f := range rp.faces {
			b := normMAC(f.BSSID)
			if b == "" {
				continue
			}
			k := faceKey{serial: rp.serial, bssid: b, mode: strings.ToLower(strings.TrimSpace(f.Mode))}
			if _, exists := faceMap[k]; exists {
				continue // already captured newer one
			}
			// create device if needed
			if _, ok := devMap[rp.serial]; !ok {
				devMap[rp.serial] = &models.Device{
					Serial: rp.serial,
					APs:    []models.Face{},
					Mesh:   []models.Face{},
				}
			}
			fo := &faceOut{
				face: models.Face{
					BSSID:     b,
					SSID:      f.SSID,
					Band:      strconv.Itoa(f.Band),
					Channel:   f.Channel,
					Mode:      k.mode,
					Clients:   nil, // fill later
					Timestamp: "",  // fill later
				},
				ts:      rp.ts,
				clients: nil,
			}
			faceMap[k] = fo
			// append to device by mode (order finalized later by sort)
			if k.mode == "ap" {
				devMap[rp.serial].APs = append(devMap[rp.serial].APs, fo.face)
			} else {
				devMap[rp.serial].Mesh = append(devMap[rp.serial].Mesh, fo.face)
			}
		}
	}

	// 4) Second pass: for the exact row that created a face, collect clients
	//    - AP faces: include associations whose station is NOT a known BSSID
	//    - Mesh faces: include associations whose station IS a known BSSID (peer mesh BSSID)
	//    Also stamp the per-face timestamp in Asia/Kolkata.
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.WithError(err).Warn("failed to load Asia/Kolkata location; using fixed offset")
		ist = time.FixedZone("IST", 5*60*60+30*60)
	}
	meshEdgeSet := map[string]models.MeshEdge{} // dedupe: from|to|ssid|band|channel

	for _, rp := range parsed {
		for _, f := range rp.faces {
			b := normMAC(f.BSSID)
			if b == "" {
				continue
			}
			mode := strings.ToLower(strings.TrimSpace(f.Mode))
			k := faceKey{serial: rp.serial, bssid: b, mode: mode}
			fo, ok := faceMap[k]
			if !ok {
				continue // this face didn't win the "latest"
			}
			if fo.ts != rp.ts {
				continue // only take clients from the exact latest row that selected this face
			}

			// collect clients according to mode
			var clients []models.FaceClient
			for _, a := range f.Associations {
				st := normMAC(a.Station)
				if st == "" {
					continue
				}
				if mode == "ap" {
					// Exclude stations that are any known BSSID (AP or mesh); only end devices remain.
					if _, isBSSID := knownBSSID[st]; isBSSID {
						continue
					}
					clients = append(clients, models.FaceClient{
						Station:       st,
						RSSI:          a.RSSI,
						Connected:     a.Connected,
						Inactive:      a.Inactive,
						RxRateBitrate: a.RxRate.Bitrate,
						TxRateBitrate: a.TxRate.Bitrate,
						RxRateChwidth: a.RxRate.Chwidth,
					})
				} else if mode == "mesh" {
					// Include only if peer is a known BSSID -> build directed mesh edge serial->peerOwner
					if _, isBSSID := knownBSSID[st]; !isBSSID {
						continue
					}
					clients = append(clients, models.FaceClient{
						Station:       st,
						RSSI:          a.RSSI,
						Connected:     a.Connected,
						Inactive:      a.Inactive,
						RxRateBitrate: a.RxRate.Bitrate,
						TxRateBitrate: a.TxRate.Bitrate,
						RxRateChwidth: a.RxRate.Chwidth,
					})
					// Directed mesh edge
					if toSerial, ok := bssidOwner[st]; ok && toSerial != "" {
						key := rp.serial + "|" + toSerial + "|" + f.SSID + "|" + strconv.Itoa(f.Band) + "|" + strconv.Itoa(f.Channel)
						if _, seen := meshEdgeSet[key]; !seen {
							meshEdgeSet[key] = models.MeshEdge{
								From:    rp.serial,
								To:      toSerial,
								SSID:    f.SSID,
								Band:    strconv.Itoa(f.Band),
								Channel: f.Channel,
							}
						}
					}
				}
			}
			// stamp timestamp in IST and set clients (nil => null)
			fo.face.Timestamp = time.Unix(fo.ts, 0).In(ist).Format(time.RFC3339)
			if len(clients) > 0 {
				// attach slice pointer to make JSON "clients":[...]
				cp := clients
				fo.clients = clients
				fo.face.Clients = &cp
			} else {
				// keep nil -> JSON null
				fo.clients = nil
				fo.face.Clients = nil
			}
		}
	}

	// 5) Move updated faces (with timestamp & clients) back into devices
	//    (We reassign the slices because fo.face was a copy in step 3)
	for serial, dev := range devMap {
		// rebuild arrays with updated faces from faceMap
		apFaces := dev.APs[:0]
		for _, f := range dev.APs {
			k := faceKey{serial: serial, bssid: f.BSSID, mode: "ap"}
			if fo, ok := faceMap[k]; ok {
				apFaces = append(apFaces, fo.face)
			}
		}
		meshFaces := dev.Mesh[:0]
		for _, f := range dev.Mesh {
			k := faceKey{serial: serial, bssid: f.BSSID, mode: "mesh"}
			if fo, ok := faceMap[k]; ok {
				meshFaces = append(meshFaces, fo.face)
			}
		}
		// sort faces for deterministic output (band asc, then bssid)
		sort.Slice(apFaces, func(i, j int) bool {
			if apFaces[i].Band == apFaces[j].Band {
				return apFaces[i].BSSID < apFaces[j].BSSID
			}
			return apFaces[i].Band < apFaces[j].Band
		})
		sort.Slice(meshFaces, func(i, j int) bool {
			if meshFaces[i].Band == meshFaces[j].Band {
				return meshFaces[i].BSSID < meshFaces[j].BSSID
			}
			return meshFaces[i].Band < meshFaces[j].Band
		})
		dev.APs = apFaces
		dev.Mesh = meshFaces
	}

	// 6) Build final ordered lists: devices by serial, mesh edges
	devs := make([]models.Device, 0, len(devMap))
	for _, d := range devMap {
		devs = append(devs, *d)
	}
	sort.Slice(devs, func(i, j int) bool { return devs[i].Serial < devs[j].Serial })

	meshEdges := make([]models.MeshEdge, 0, len(meshEdgeSet))
	for _, e := range meshEdgeSet {
		meshEdges = append(meshEdges, e)
	}
	sort.Slice(meshEdges, func(i, j int) bool {
		if meshEdges[i].From == meshEdges[j].From {
			if meshEdges[i].To == meshEdges[j].To {
				if meshEdges[i].SSID == meshEdges[j].SSID {
					if meshEdges[i].Band == meshEdges[j].Band {
						return meshEdges[i].Channel < meshEdges[j].Channel
					}
					return meshEdges[i].Band < meshEdges[j].Band
				}
				return meshEdges[i].SSID < meshEdges[j].SSID
			}
			return meshEdges[i].To < meshEdges[j].To
		}
		return meshEdges[i].From < meshEdges[j].From
	})

	// 7) Final response
	out := models.Topology{
		BoardID:   boardID,
		Timestamp: time.Unix(end, 0).UTC().Format(time.RFC3339),
		Nodes:     devs,
		Edges:     models.TopoEdges{Wired: []any{}, Mesh: meshEdges},
		External:  []any{},
	}
	if log != nil {
		log.WithFields(logger.Fields{
			"nodes":      len(out.Nodes),
			"mesh_edges": len(out.Edges.Mesh),
		}).Trace("topology built successfully")
	}
	return out, nil
}

func normMAC(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func StringPtr(s string) *string {
	return &s
}

func IntrPtr(i int) *int {
	return &i
}
