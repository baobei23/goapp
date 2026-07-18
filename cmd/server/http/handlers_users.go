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

// changePassword godoc
//
//	@Summary		Change Password
//	@Description	Change Password
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			body	body		object{old_password=string,new_password=string}	true	"Passwords"
//	@Success		200		{object}	BaseResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/users/password [put]
//
//	@security		ApiKeyAuth
func (h *Handlers) ChangePassword(c *gin.Context) {
	email := GetUserEmail(c)
	if email == "" {
		Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, err)
		return
	}

	if err := h.apis.ChangePassword(c.Request.Context(), email, req.OldPassword, req.NewPassword); err != nil {
		if err.Error() == "invalid credentials" {
			Error(c, http.StatusUnauthorized, err)
			return
		}
		Error(c, http.StatusInternalServerError, err)
		return
	}

	JSON(c, http.StatusOK, map[string]string{"message": "password updated successfully"}, nil)
}
