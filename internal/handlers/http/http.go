package http

import (
	"context"
	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Handler struct {
	logger *zap.SugaredLogger
	app    App
}

func New(app App, logger *zap.SugaredLogger) *Handler {
	return &Handler{
		app:    app,
		logger: logger,
	}
}

type App interface {
	GetBalance(context.Context, string) (string, error)
	HealthCheck(context.Context) error
}

func (h Handler) Start(port string, timeoutLimit time.Duration) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(timeout.New(
		timeout.WithTimeout(timeoutLimit),
		timeout.WithHandler(func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(c.Request.Context(), timeoutLimit)
			defer cancel()
			c.Request.WithContext(ctx)
			c.Next()
		}),
		timeout.WithResponse(func(c *gin.Context) {
			c.JSON(http.StatusRequestTimeout, gin.H{
				"message": "Request timeout",
			})
		}),
	))
	r.GET("/eth/balance/:address", h.Balance())

	r.GET("/metrics", h.Metrics())
	r.GET("/healthcheck", h.HealthCheck())

	return r.Run(":" + port)
}

func (h Handler) Balance() gin.HandlerFunc {
	return func(c *gin.Context) {
		address := c.Param("address")
		if address == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"message": "no address specified",
			})
			return
		}
		balance, err := h.app.GetBalance(c.Request.Context(), address)
		if err != nil {
			h.logger.Errorw("failed to get balance", "address", address, "error", err)

			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"message": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"balance": balance,
		})
	}
}

func (h Handler) HealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.app.HealthCheck(c.Request.Context()); err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	}
}

func (h Handler) Metrics() gin.HandlerFunc {
	p := promhttp.Handler()

	return func(c *gin.Context) {
		p.ServeHTTP(c.Writer, c.Request)
	}
}
