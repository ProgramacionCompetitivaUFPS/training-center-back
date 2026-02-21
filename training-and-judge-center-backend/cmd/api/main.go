package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/training-judge-center/backend/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := server.NewRouter()

	slog.Info("server starting", "port", port)

	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), router); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
