package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog/log"
)

func main() {
	log.Debug().Msgf("%d redirects configured", len(config.Redirects))

	for _, r := range config.Redirects {
		log.Debug().Str("from", r.From).Str("to", r.To).Int("status", r.Status).Send()
	}

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	mux := http.NewServeMux()

	mux.HandleFunc("/__health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		for _, redirect := range config.Redirects {
			if r.Host != redirect.From {
				continue
			}

			log.Debug().Str("method", r.Method).Str("host", redirect.From).Str("redirect", redirect.To).Send()

			destination := redirect.To + r.URL.RequestURI()
			http.Redirect(w, r, destination, redirect.Status)
			return
		}

		log.Debug().Str("method", r.Method).Str("host", r.Host).Send()
		http.NotFound(w, r)
	})

	server := http.Server{
		Addr:    config.ServerAddress,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("server error")
		}
	}()

	<-ctx.Done()

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx, cancel = signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	log.Debug().Msg("gracefully shutting down")

	go func() {
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("server close error")
		}
	}()

	<-ctx.Done()

	if ctx.Err() == context.DeadlineExceeded {
		log.Warn().Msg("server shutdown timed out")
	} else {
		log.Debug().Msg("goodbye")
	}
}
