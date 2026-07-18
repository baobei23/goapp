package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// readUserByEmail godoc
//
//	@Summary		Read User By Email
//	@Description	Read User By Email
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	BaseResponse{data=users.User}
//	@Failure		400	{object}	ErrorResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Router			/users [get]
//
//	@security		ApiKeyAuth
func (h *Handlers) ReadUserByEmail(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	out, err := h.apis.ReadUserByEmail(c.Request.Context(), email)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	JSON(c, http.StatusOK, out, nil)
}
