package main

import (
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	riq "github.com/libops/riq/internal/config"
)

func main() {
	config, err := riq.ReadConfig("riq.yaml")
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	for _, middleware := range config.Queues {
		numConsumers := middleware.Consumers
		for i := range numConsumers {
			wg.Add(1)
			go func(middleware riq.Queue, consumerID int) {
				defer wg.Done()

				slog.Info("Starting subscriber", "queue", middleware.Name, "consumer", consumerID)

				for {
					select {
					case <-stopChan:
						slog.Info("Stopping subscriber", "queue", middleware.Name, "consumer", consumerID)
						return
					default:
						err := middleware.RecvAndProcessMessage()
						if err != nil {
							slog.Error("Error processing message", "queue", middleware.Name, "consumer", consumerID, "error", err)
						}
					}
				}
			}(middleware, i)
		}
	}

	<-stopChan
	slog.Info("Shutting down all message listeners")
	wg.Wait()
}
