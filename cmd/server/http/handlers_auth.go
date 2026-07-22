package http

import (
	"errors"
	"net/http"
	"time"

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
	AccessToken string      `json:"accessToken"`
	ExpiresIn   int64       `json:"expiresIn"`
	User        *users.User `json:"user"`
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

	accessToken, refreshToken, jti, err := h.tm.GeneratePair(user.ID, user.Email)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	err = h.apis.SaveRefreshToken(c.Request.Context(), jti, user.ID, time.Now().Add(h.tm.GetRefreshExpiry()))
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	c.SetCookie("refreshToken", refreshToken, int(h.tm.GetRefreshExpiry().Seconds()), "/", "", false, true)

	JSON(c, http.StatusOK, &LoginResponse{
		AccessToken: accessToken,
		ExpiresIn:   int64(h.tm.GetAccessExpiry().Seconds()),
		User:        user,
	}, nil)
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}
type RefreshTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
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
	_ = c.ShouldBindJSON(&req)

	token := req.RefreshToken
	if token == "" {
		token, _ = c.Cookie("refreshToken")
	}
	if token == "" {
		Error(c, http.StatusBadRequest, errors.New("refresh token required"))
		return
	}

	claims, err := h.tm.Validate(token)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	if claims.TokenType != "refresh" {
		Error(c, http.StatusUnauthorized, errors.New("invalid token type"))
		return
	}

	exists, err := h.apis.CheckRefreshToken(c.Request.Context(), claims.ID)
	if err != nil || !exists {
		Error(c, http.StatusUnauthorized, errors.New("refresh token invalid or revoked"))
		return
	}

	_ = h.apis.RevokeRefreshToken(c.Request.Context(), claims.ID)

	accessToken, refreshToken, newJti, err := h.tm.GeneratePair(claims.UserID, claims.Email)
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	err = h.apis.SaveRefreshToken(c.Request.Context(), newJti, claims.UserID, time.Now().Add(h.tm.GetRefreshExpiry()))
	if err != nil {
		Error(c, http.StatusInternalServerError, err)
		return
	}

	c.SetCookie("refreshToken", refreshToken, int(h.tm.GetRefreshExpiry().Seconds()), "/", "", false, true)

	JSON(c, http.StatusOK, &RefreshTokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   int64(h.tm.GetAccessExpiry().Seconds()),
	}, nil)
}

// logout godoc
//
//	@Summary		Logout
//	@Description	Logout by revoking refresh token
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		RefreshTokenRequest	true	"Refresh Token Payload"
//	@Success		200		{object}	BaseResponse{data=string}
//	@Failure		400		{object}	ErrorResponse
//	@Router			/auth/logout [post]
func (h *Handlers) Logout(c *gin.Context) {
	req := &RefreshTokenRequest{}
	_ = c.ShouldBindJSON(&req)

	token := req.RefreshToken
	if token == "" {
		token, _ = c.Cookie("refreshToken")
	}

	c.SetCookie("refreshToken", "", -1, "/", "", false, true)

	var loggedOut string = "logged out"

	if token == "" {
		JSON(c, http.StatusOK, loggedOut, nil)
		return
	}

	claims, err := h.tm.Validate(token)
	if err != nil || claims.TokenType != "refresh" {
		// Ignore validation errors on logout
		JSON(c, http.StatusOK, loggedOut, nil)
		return
	}

	_ = h.apis.RevokeRefreshToken(c.Request.Context(), claims.ID)

	JSON(c, http.StatusOK, loggedOut, nil)
}
