package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/router-architects/ra-openlan-nw-topology/internal/apperrors"
	"github.com/router-architects/ra-openlan-nw-topology/internal/logger"
	"github.com/router-architects/ra-openlan-nw-topology/internal/models"
	"github.com/router-architects/ra-openlan-nw-topology/internal/services"
)

type TopologyHandler struct {
	svc services.TopologyService
}

func NewTopologyHandler(s services.TopologyService) *TopologyHandler {
	return &TopologyHandler{svc: s}
}

// GET /v1/topology?boardId={id}&at={RFC3339 optional}
func (h *TopologyHandler) GetTopology(c fiber.Ctx) error {
	log := logger.ForFunctionality("TOPOLOGY-HANDLER")
	params := &models.TimepointsQuery{}
	if err := c.Bind().Query(params); err != nil {
		if log != nil {
			log.WithError(err).Warn("failed to bind topology query params")
		}
		return writeErrorResponse(c, apperrors.CodeInvalidInput)
	}

	topo, err := h.svc.BuildTopology(c.Context(), params.BoardID, *params)
	if err != nil {
		appErr, ok := err.(*apperrors.Error)
		if !ok {
			return writeErrorResponse(c, apperrors.CodeInternal)
		}
		if log != nil {
			log.WithError(err).
				WithField("errorCode", appErr.Code).
				Error("topology build failed")
		}
		return writeErrorResponse(c, appErr.Code)
	}
	return c.Status(fiber.StatusOK).JSON(topo)
}

func writeErrorResponse(c fiber.Ctx, code apperrors.ErrorCode) error {
	info := apperrors.GetHTTPErrorInfo(code)
	body := map[string]any{
		"ErrorCode":        info.Status,
		"ErrorDescription": fmt.Sprintf("%d: %s", info.Status, info.Description),
		"ErrorDetails":     c.Method(),
	}
	return c.Status(info.Status).JSON(body)
}
