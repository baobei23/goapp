package http

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/baobei23/goapp/internal/api"
	"github.com/baobei23/goapp/internal/pkg/jwt"
)

// Handlers struct has all the dependencies required for HTTP handlers
type Handlers struct {
	apis api.Server
	home *template.Template
	tm   *jwt.TokenManager
}

func (h *Handlers) registerRoutes(r *gin.Engine) {

	//Documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	//root
	r.GET("/", h.HelloWorld)

	//auth
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/auth/refresh", h.RefreshToken)

	protected := r.Group("/")
	protected.Use(h.AuthMiddleware())

	//users
	protected.GET("/users", h.ReadUserByEmail)
	protected.PUT("/users/password", h.ChangePassword)

	//usernotes
	protected.POST("/usernotes", h.RegisterNote)
	protected.GET("/usernotes/:noteID", h.ReadUserNote)
}

func (h *Handlers) HelloWorld(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")
	switch contentType {
	case "application/json":
		c.JSON(http.StatusOK, "hello world")
	default:
		buff := bytes.NewBufferString("")
		err := h.home.Execute(
			buff,
			struct {
				Message string
			}{
				Message: "Welcome to the Home Page!",
			},
		)
		if err != nil {
			Error(c, http.StatusInternalServerError, err)
			return
		}

		c.Header("Content-Type", "text/html; charset=UTF-8")
		c.String(http.StatusOK, buff.String())
	}
}

func loadHomeTemplate(basePath string) (*template.Template, error) {
	t := template.New("index.html")
	home, err := t.ParseFiles(
		fmt.Sprintf("%s/index.html", basePath),
	)
	if err != nil {
		return nil, fmt.Errorf("failed parsing templates: %w", err)
	}

	return home, nil
}
