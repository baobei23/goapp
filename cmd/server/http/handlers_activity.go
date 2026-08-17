package http

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/baobei23/goapp/internal/events"
)

// GetActivity godoc
//
//	@Summary		Get user activity log
//	@Description	Returns recent activity events for the authenticated user (sourced from Kafka consumer)
//	@Tags			Activity
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int	false	"Max events to return (default 50, max 100)"
//	@Success		200		{object}	BaseResponse{data=[]events.Activity}
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/activity [get]
//	@Security		ApiKeyAuth
func (h *Handlers) GetActivity(c *gin.Context) {
	userID := GetUserID(c)
	if userID == "" {
		Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	activities, err := h.apis.GetUserActivity(c.Request.Context(), userID, limit)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}
	if activities == nil {
		activities = []events.Activity{}
	}

	JSON(c, http.StatusOK, activities, nil)
}
