package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naughtygopher/errors"
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
func (h *Handlers) ReadUserByEmail(c *gin.Context) error {
	email := GetUserEmail(c)
	if email == "" {
		return errors.Unauthorized("unauthorized")
	}

	out, err := h.apis.ReadUserByEmail(c.Request.Context(), email)
	if err != nil {
		return err
	}

	JSON(c, http.StatusOK, out, nil)

	return nil
}
