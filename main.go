package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/libops/riq/internal/config"
	"github.com/libops/riq/internal/stomp"
)

func main() {
	config, err := config.ReadConfig("riq.yaml")
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	for _, middleware := range config.Queues {
		numConsumers := middleware.Consumers
		for i := range numConsumers {
			wg.Add(1)
			go func(ctx context.Context, middleware stomp.Queue, consumerID int) {
				defer wg.Done()

				slog.Info("Starting subscriber", "queue", middleware.Name, "consumer", consumerID)

				for {
					select {
					case <-stopChan:
						slog.Info("Stopping subscriber", "queue", middleware.Name, "consumer", consumerID)
						return
					default:
						err := middleware.RecvAndProcessMessage(ctx)
						if err != nil {
							slog.Error("Error processing message", "queue", middleware.Name, "consumer", consumerID, "error", err)
						}
					}
				}
			}(ctx, middleware, i)
		}
	}

	<-stopChan
	slog.Info("All stomp subscribers are now running")
	wg.Wait()
	slog.Info("All stomp subscribers have stopped")
}
