package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/steipete/wacli/internal/app"
)

type Server struct {
	Router *gin.Engine
	App    *app.App
	Config *Config
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.App != nil {
		s.App.Close()
	}
	return nil
}
