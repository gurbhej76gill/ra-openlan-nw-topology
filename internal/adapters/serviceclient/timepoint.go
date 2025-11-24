package serviceclient

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/router-architects/ra-openlan-nw-topology/internal/apperrors"
	"github.com/router-architects/ra-openlan-nw-topology/internal/logger"
	"github.com/router-architects/ra-openlan-nw-topology/internal/models"
)

func (v *OpenAPIRequest) GetTimepoints(ctx context.Context, req models.TimepointRequest) ([]models.TimepointRow, error) {
	fullURL := "/api/v1/board"
	if req.BoardID != "" {
		fullURL += "/" + req.BoardID
	} else {
		return nil, apperrors.WrapError(apperrors.CodeInvalidInput, "boardId is required", nil)
	}

	fullURL += "/timepoints?"

	if req.FromDate != nil {
		fullURL += "fromDate=" + *req.FromDate + "&"
	}
	if req.EndDate != nil {
		fullURL += "endDate=" + *req.EndDate + "&"
	}
	if req.MaxRecords != nil {
		fullURL += "maxRecords=" + strconv.Itoa(*req.MaxRecords) + "&"
	}
	if req.StatsOnly {
		fullURL += "statsOnly=true&"
	}
	if req.PointsOnly {
		fullURL += "pointsOnly=true&"
	}
	if req.PointStatsOnly {
		fullURL += "pointStatsOnly=true"
	}

	logFields := logger.Fields{
		"boardId":        req.BoardID,
		"statsOnly":      req.StatsOnly,
		"pointsOnly":     req.PointsOnly,
		"pointStatsOnly": req.PointStatsOnly,
	}
	if req.FromDate != nil {
		logFields["fromDate"] = *req.FromDate
	}
	if req.EndDate != nil {
		logFields["endDate"] = *req.EndDate
	}
	if req.MaxRecords != nil {
		logFields["maxRecords"] = *req.MaxRecords
	}
	log := logger.ForFunctionality("TIMEPOINTS-CLIENT").WithFields(logFields)
	start := time.Now()

	resp, err := v.Do(ctx, fiber.MethodGet, "owanalytics", fullURL, nil)

	if err != nil {
		return nil, apperrors.WrapError(apperrors.CodeInternal, "failed to get timepoints", err)
	}

	if resp.StatusCode() == fiber.StatusNotFound {
		log.WithField("status", resp.StatusCode()).Error("timepoints not found")
		info := apperrors.GetHTTPErrorInfo(apperrors.CodeNotFound)
		return nil, apperrors.WrapError(apperrors.CodeNotFound, info.Description, nil)
	}

	if resp.StatusCode() != fiber.StatusOK {
		return nil, apperrors.WrapError(apperrors.CodeInternal, "failed to get timepoints: non-200 response", nil)
	}

	type timepointResponse struct {
		Points [][]models.TimepointRow `json:"points"`
	}
	var tpResp timepointResponse
	if err := json.Unmarshal(resp.Body(), &tpResp); err != nil {
		return nil, apperrors.WrapError(apperrors.CodeInternal, "failed to parse timepoints response", err)
	}

	var timepoints []models.TimepointRow
	for _, bucket := range tpResp.Points {
		if len(bucket) == 0 {
			continue
		}
		timepoints = append(timepoints, bucket...)
	}

	log.WithFields(logger.Fields{
		"records":     len(timepoints),
		"status":      resp.StatusCode(),
		"duration_ms": time.Since(start).Milliseconds(),
	}).Trace("received timepoints response")

	return timepoints, nil

}
