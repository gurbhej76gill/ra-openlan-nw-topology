package repositories

import (
	"context"
	"errors"

	"github.com/router-architects/ra-openlan-nw-topology/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TopologyRepository interface {
	LatestTimestamp(ctx context.Context, boardID string) (int64, error)
	FetchTimepoints(ctx context.Context, boardID string, start, end int64) ([]models.TimepointRowDB, error)
}

type pgRepo struct {
	pool *pgxpool.Pool
}

func NewTopologyRepository(pool *pgxpool.Pool) TopologyRepository {
	return &pgRepo{pool: pool}
}

func (r *pgRepo) LatestTimestamp(ctx context.Context, boardID string) (int64, error) {
	const q = `SELECT COALESCE(MAX("timestamp"), 0) FROM public.timepoints WHERE boardid = $1`
	var maxTs int64
	if err := r.pool.QueryRow(ctx, q, boardID).Scan(&maxTs); err != nil {
		return 0, err
	}
	if maxTs == 0 {
		return 0, errors.New("no_timestamp_for_board")
	}
	return maxTs, nil
}

func (r *pgRepo) FetchTimepoints(ctx context.Context, boardID string, start, end int64) ([]models.TimepointRowDB, error) {
	const q = `
SELECT id, boardid, "timestamp", ssid_data, device_info, serialnumber
FROM public.timepoints
WHERE boardid = $1 AND "timestamp" > $2 AND "timestamp" <= $3
ORDER BY "timestamp" DESC`
	rows, err := r.pool.Query(ctx, q, boardID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.TimepointRowDB
	for rows.Next() {
		var rID, rBoard, rSSID, rDev, rSerial string
		var rTs int64
		if err := rows.Scan(&rID, &rBoard, &rTs, &rSSID, &rDev, &rSerial); err != nil {
			return nil, err
		}
		out = append(out, models.TimepointRowDB{
			ID:         rID,
			BoardID:    rBoard,
			Timestamp:  rTs,
			SSIDData:   rSSID,
			DeviceInfo: rDev,
			Serial:     rSerial,
		})
	}
	return out, rows.Err()
}
