package httpserver

import (
	"time"
)

type Option func(*Server)

func Port(port int) Option {
	return func(s *Server) {
		s.port = port
	}
}

func Prefork(prefork bool) Option {
	return func(s *Server) {
		s.prefork = prefork
	}
}

func ReadTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.readTimeout = timeout
	}
}

func WriteTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.writeTimeout = timeout
	}
}

func ShutDownTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.shutdownTimeout = timeout
	}
}
