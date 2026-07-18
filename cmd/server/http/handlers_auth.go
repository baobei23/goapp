package http

import (
	"errors"
	"net/http"

	"github.com/baobei23/goapp/internal/users"
	"github.com/gin-gonic/gin"
)

type RegisterRequest struct {
	FullName string `json:"fullName" binding:"required,max=255"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=8"`
}

// register godoc
//
//	@Summary		Register a new user
//	@Description	Register a new user
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		RegisterRequest	true	"Register Payload"
//	@Success		201		{object}	BaseResponse{data=users.User}
//	@Failure		400		{object}	ErrorResponse
//	@Failure		409		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/register [post]
func (h *Handlers) Register(c *gin.Context) {
	req := &RegisterRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, err)
		return
	}

	u := &users.User{
		FullName: req.FullName,
		Email:    req.Email,
		Password: []byte(req.Password),
	}

	createdUser, err := h.apis.Register(c.Request.Context(), u)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	JSON(c, http.StatusCreated, createdUser, nil)
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
type LoginResponse struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	ExpiresIn    int64       `json:"expiresIn"`
	User         *users.User `json:"user"`
}

// login godoc
//
//	@Summary		Login
//	@Description	Login
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		LoginRequest	true	"Login Payload"
//	@Success		200		{object}	BaseResponse{data=LoginResponse}
//	@Failure		400		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/login [post]
func (h *Handlers) Login(c *gin.Context) {
	req := &LoginRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := h.apis.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	accessToken, refreshToken, err := h.tm.GeneratePair(user.ID, user.Email)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	JSON(c, http.StatusOK, &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.tm.GetAccessExpiry().Seconds()),
		User:         user,
	}, nil)
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}
type RefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

// refreshToken godoc
//
//	@Summary		Refresh Access Token
//	@Description	Use valid refresh token to get new access token pair
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		RefreshTokenRequest	true	"Refresh Token Payload"
//	@Success		200		{object}	BaseResponse{data=RefreshTokenResponse}
//	@Failure		400		{object}	ErrorResponse
//	@Failure		500		{object}	ErrorResponse
//	@Router			/auth/refresh [post]
func (h *Handlers) RefreshToken(c *gin.Context) {
	req := &RefreshTokenRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, err)
		return
	}

	claims, err := h.tm.Validate(req.RefreshToken)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	if claims.TokenType != "refresh" {
		Error(c, http.StatusUnauthorized, errors.New("invalid token type"))
		return
	}

	accessToken, refreshToken, err := h.tm.GeneratePair(claims.UserID, claims.Email)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	JSON(c, http.StatusOK, &RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(h.tm.GetAccessExpiry().Seconds()),
	}, nil)
}
