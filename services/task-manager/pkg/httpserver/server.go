package httpserver

import (
	"app/pkg/logger"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

const (
	_defaultPort            = 8080
	_defaultReadTimeout     = 60 * time.Second
	_defaultWriteTimeout    = 60 * time.Second
	_defaultShutdownTimeout = 30 * time.Second
)

// Server представляет собой http сервер
type Server struct {
	ctx             context.Context
	eg              *errgroup.Group
	engine          *gin.Engine
	server          *http.Server
	notify          chan error
	port            int
	prefork         bool
	readTimeout     time.Duration
	writeTimeout    time.Duration
	shutdownTimeout time.Duration
	l               logger.Interface
}

// New создает новый сервер
func New(l logger.Interface, opts ...Option) *Server {
	group, ctx := errgroup.WithContext(context.Background())

	group.SetLimit(1)

	gin.SetMode(gin.ReleaseMode)

	s := &Server{
		ctx: ctx,
		eg:  group,

		notify: make(chan error, 1),

		port:            _defaultPort,
		prefork:         false,
		readTimeout:     _defaultReadTimeout,
		writeTimeout:    _defaultWriteTimeout,
		shutdownTimeout: _defaultShutdownTimeout,
		l:               l,
	}

	//Применяем опции
	for _, opt := range opts {
		opt(s)
	}

	//Создаем движок gin
	engine := gin.New()
	s.engine = engine

	//Создаем http сервер
	s.server = &http.Server{
		Addr:           fmt.Sprintf(":%d", s.port),
		Handler:        engine,
		ReadTimeout:    s.readTimeout,
		WriteTimeout:   s.writeTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	return s
}

func (s *Server) Start() {
	s.eg.Go(func() error {
		s.l.Info("starting http server", "port", s.port)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.notify <- err
			close(s.notify)
			s.l.Error("failed to listen and serve", "error", err)
			return err
		}
		return nil
	})
}

func (s *Server) Notify() <-chan error {
	return s.notify
}

func (s *Server) Shutdown() error {
	var shutdownErrors []error

	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
		s.l.Error("server shutdown error", "error", err)
		shutdownErrors = append(shutdownErrors, err)
	}

	if err := s.eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		s.l.Error("errgroup wait error", "error", err)
		shutdownErrors = append(shutdownErrors, err)
	}

	s.l.Info("HTTP server stopped")
	return errors.Join(shutdownErrors...)
}

func (s *Server) Engine() *gin.Engine {
	return s.engine
}
