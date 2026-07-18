package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/baobei23/goapp/internal/usernotes"
)

type RegisterNoteRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// createNote godoc
//
//	@Summary		Create User Note
//	@Description	Create a new note for the authenticated user
//	@Tags			Notes
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		RegisterNoteRequest	true	"Note Payload"
//	@Success		201		{object}	BaseResponse{data=RegisterNoteRequest}
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/usernotes [post]
//	@Security		ApiKeyAuth
func (h *Handlers) RegisterNote(c *gin.Context) {
	userID := GetUserID(c)
	if userID == "" {
		Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	req := &RegisterNoteRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		Error(c, http.StatusBadRequest, err)
		return
	}

	unote := &usernotes.Note{
		Title:   req.Title,
		Content: req.Content,
		UserID:  userID,
	}

	un, err := h.apis.RegisterNote(c.Request.Context(), unote)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	JSON(c, http.StatusCreated, un, nil)
}

// readUserNote godoc
//
//	@Summary		Read User Note
//	@Description	Read a user note
//	@Tags			Notes
//	@Accept			json
//	@Produce		json
//	@Param			noteID	path		string	true	"Note ID"
//	@Success		200		{object}	BaseResponse{data=usernotes.Note}
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/usernotes/{noteID} [get]
//	@Security		ApiKeyAuth
func (h *Handlers) ReadUserNote(c *gin.Context) {
	userID := GetUserID(c)
	if userID == "" {
		Error(c, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	noteID := c.Param("noteID")
	if noteID == "" {
		Error(c, http.StatusBadRequest, errors.New("noteID is required"))
		return
	}

	un, err := h.apis.ReadUserNote(c.Request.Context(), userID, noteID)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	JSON(c, http.StatusOK, un, nil)
}
