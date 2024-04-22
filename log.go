package main

import (
	"log/slog"
	"os"
)

var (
	log = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)
