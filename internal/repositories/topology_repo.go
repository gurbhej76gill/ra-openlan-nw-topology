package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TopologyRepository interface {
	FetchLatestRows(ctx context.Context, groupID string, start, end time.Time) ([]row, error)
}

type repo struct {
	pool *pgxpool.Pool
}

func NewTopologyRepo(pool *pgxpool.Pool) TopologyRepository {
	return &repo{pool: pool}
}

type row struct {
	id           string
	boardID      string
	timestamp    time.Time
	serialNumber string
	ssidData     map[string]any
	deviceInfo   map[string]any
}

const q = `
SELECT DISTINCT ON (serialnumber)
  id, boardid, timestamp, serialnumber, ssid_data, device_info
FROM timepoints
WHERE boardid = $1
  AND timestamp BETWEEN $2 AND $3
ORDER BY serialnumber, timestamp DESC;
`

func (r *repo) FetchLatestRows(ctx context.Context, groupID string, start, end time.Time) ([]row, error) {
	rows, err := r.pool.Query(ctx, q, groupID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query latest rows: %w", err)
	}
	defer rows.Close()

	var out []row
	for rows.Next() {
		var rr row
		var ssidRaw, infoRaw []byte
		if err := rows.Scan(&rr.id, &rr.boardID, &rr.timestamp, &rr.serialNumber, &ssidRaw, &infoRaw); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if len(ssidRaw) > 0 {
			if err := json.Unmarshal(ssidRaw, &rr.ssidData); err != nil {
				return nil, fmt.Errorf("unmarshal ssid_data: %w", err)
			}
		}
		if len(infoRaw) > 0 {
			if err := json.Unmarshal(infoRaw, &rr.deviceInfo); err != nil {
				return nil, fmt.Errorf("unmarshal device_info: %w", err)
			}
		}
		out = append(out, rr)
	}
	if err := rows.Err(); err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("rows err: %w", err)
	}
	return out, nil
}
