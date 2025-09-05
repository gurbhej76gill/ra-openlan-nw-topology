package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/router-architects/network-topology/internal/apperrors"
	"github.com/router-architects/network-topology/internal/services"
)

type topologyHandler struct {
	svc services.TopologyService
}

func NewTopologyHandler(svc services.TopologyService) *topologyHandler {
	return &topologyHandler{svc: svc}
}

func (h *topologyHandler) Register(gr fiber.Router) {
	gr.Get("/topology", h.getTopology)
}

func (h *topologyHandler) getTopology(c fiber.Ctx) error {
	groupID := c.Query("groupId")
	if groupID == "" {
		return h.badRequest(c, "groupId is required")
	}
	if _, err := uuid.Parse(groupID); err != nil {
		return h.badRequest(c, "groupId must be a valid UUID")
	}

	atStr := c.Query("at")
	var at time.Time
	var err error
	if atStr == "" {
		at = time.Now().UTC()
	} else {
		at, err = time.Parse(time.RFC3339, atStr)
		if err != nil {
			return h.badRequest(c, "at must be RFC3339 timestamp")
		}
	}

	logrus.WithFields(logrus.Fields{
		"groupId": groupID,
		"at":      at.Format(time.RFC3339),
	}).Debug("handling topology request")

	resp, err := h.svc.Build(c.Context(), groupID, at)
	if err != nil {
		logrus.WithError(err).Error("build topology failed")
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": apperrors.ErrInternal.Error(), "message": "failed to build topology"})
	}

	// No pagination headers since this is not a list endpoint
	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *topologyHandler) badRequest(c fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error":   apperrors.ErrBadRequest.Error(),
		"message": msg,
	})
}
