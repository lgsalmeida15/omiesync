package etl

import (
	"net/http/httptest"

	"github.com/rs/zerolog"

	"omie-sync-api/internal/omie"
)

func nopLogger() zerolog.Logger { return zerolog.Nop() }

func clientForServer(srv *httptest.Server) *omie.Client {
	c := omie.NewClient("test-key", "test-secret")
	c.SetBaseURL(srv.URL)
	return c
}
