package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/router-architects/network-topology-service/internal/apperrors"
	"github.com/router-architects/network-topology-service/internal/services"
)

type TopologyHandler struct {
	svc services.TopologyService
}

func NewTopologyHandler(s services.TopologyService) *TopologyHandler {
	return &TopologyHandler{svc: s}
}

// GET /v1/topology?boardId={id}&at={RFC3339 optional}
func (h *TopologyHandler) GetTopology(c fiber.Ctx) error {
	boardID := c.Query("boardId")
	if boardID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(errorBody(apperrors.CodeInvalidInput, "boardId is required"))
	}
	var atPtr *time.Time
	if at := c.Query("at"); at != "" {
		// missing zone -> UTC
		// Parse strictly: try RFC3339, else try "2006-01-02T15:04:05" as UTC.
		if t, err := time.Parse(time.RFC3339, at); err == nil {
			ut := t.UTC()
			atPtr = &ut
		} else if t2, err2 := time.Parse("2006-01-02T15:04:05", at); err2 == nil {
			ut := t2.UTC()
			atPtr = &ut
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(errorBody(apperrors.CodeInvalidInput, "invalid 'at' format; use RFC3339 or YYYY-MM-DDTHH:MM:SS"))
		}
	}
	topo, err := h.svc.BuildTopology(c.Context(), boardID, atPtr, c.Locals("topology_window").(time.Duration), c.Locals("topology_drift").(time.Duration))
	if err != nil {
		// internal errors must not leak
		return c.Status(fiber.StatusInternalServerError).JSON(errorBody(apperrors.CodeInternal, "failed to build topology"))
	}
	return c.Status(fiber.StatusOK).JSON(topo)
}

func errorBody(code apperrors.ErrorCode, msg string) map[string]any {
	return map[string]any{"error": map[string]any{
		"code":    code,
		"message": msg,
	}}
}
