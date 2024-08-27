package main

import (
	"news/src/cmd"
	"news/src/logger"
)

func main() {
	logger.Info("Starting server...")

	cmd.StartAPIServer()

	logger.Info("Server started.")
}
