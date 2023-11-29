package beacon

import (
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/ledgerwatch/erigon/cl/beacon/beacon_router_configuration"
	"github.com/ledgerwatch/erigon/cl/beacon/handler"
	"github.com/ledgerwatch/erigon/cl/beacon/validatorapi"
	"github.com/ledgerwatch/log/v3"
)

type LayeredBeaconHandler struct {
	ValidatorApi *validatorapi.ValidatorApiHandler
	ArchiveApi   *handler.ApiHandler
}

func ListenAndServe(beaconHandler *LayeredBeaconHandler, routerCfg beacon_router_configuration.RouterConfiguration) {
	listener, err := net.Listen(routerCfg.Protocol, routerCfg.Address)
	if err != nil {
		panic(err)
	}

	mux := chi.NewRouter()
	// enforce json content type
	mux.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			if len(contentType) > 0 && !strings.EqualFold(contentType, "application/json") {
				http.Error(w, "Content-Type header must be application/json", http.StatusUnsupportedMediaType)
				return
			}
			h.ServeHTTP(w, r)
		})
	})
	// layered handling - 404 on first handler falls back to the second
	mux.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {

		nfw := &notFoundNoWriter{rw: w}
		beaconHandler.ValidatorApi.ServeHTTP(nfw, r)
		if nfw.code == 404 || nfw.code == 0 {
			beaconHandler.ArchiveApi.ServeHTTP(w, r)
		}
	})
	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  routerCfg.ReadTimeTimeout,
		IdleTimeout:  routerCfg.IdleTimeout,
		WriteTimeout: routerCfg.IdleTimeout,
	}
	if err != nil {
		log.Warn("[Beacon API] Failed to start listening", "addr", routerCfg.Address, "err", err)
	}

	if err := server.Serve(listener); err != nil {
		log.Warn("[Beacon API] failed to start serving", "addr", routerCfg.Address, "err", err)
	}
}

func newBeaconMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" && contentType != "" {
			http.Error(w, "Content-Type header must be application/json", http.StatusUnsupportedMediaType)
			return
		}
		next.ServeHTTP(w, r)
	})
}
