package http

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naughtygopher/errors"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/baobei23/goapp/internal/api"
	"github.com/baobei23/goapp/internal/pkg/jwt"
	"github.com/baobei23/goapp/internal/pkg/logger"
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
	r.GET("/", errWrapper(h.HelloWorld))

	//auth
	r.POST("/register", errWrapper(h.Register))
	r.POST("/login", errWrapper(h.Login))
	r.POST("/auth/refresh", errWrapper(h.RefreshToken))

	protected := r.Group("/")
	protected.Use(h.AuthMiddleware())

	//users
	protected.GET("/users", errWrapper(h.ReadUserByEmail))

	//usernotes
	protected.POST("/usernotes", errWrapper(h.RegisterNote))
	protected.GET("/usernotes/:noteID", errWrapper(h.ReadUserNote))
}

func (h *Handlers) HelloWorld(c *gin.Context) error {
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
			return errors.InternalErr(err, "Inter server error")
		}

		c.Header("Content-Type", "text/html; charset=UTF-8")
		c.String(http.StatusOK, buff.String())
	}
	return nil
}

func errWrapper(h func(c *gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := h(c)
		if err == nil {
			return
		}

		status, msg, _ := errors.HTTPStatusCodeMessage(err)
		Error(c, status, fmt.Errorf("%s", msg))
		if status > 499 {
			logger.Error(c.Request.Context(), errors.Stacktrace(err))
		}
	}
}

func loadHomeTemplate(basePath string) (*template.Template, error) {
	t := template.New("index.html")
	home, err := t.ParseFiles(
		fmt.Sprintf("%s/index.html", basePath),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing templates")
	}

	return home, nil
}
