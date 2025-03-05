package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/libops/isle-event-bus/internal/config"
	"github.com/libops/isle-event-bus/internal/stomp"
)

func main() {
	config, err := config.ReadConfig("isle-event-bus.yaml")
	if err != nil {
		slog.Error("Could not read YML", "err", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	for _, q := range config.Queues {
		if q.Disabled {
			slog.Info("Skipping disabled subscriber", "queue", q.Name)
			continue
		}
		numConsumers := q.Consumers
		for i := range numConsumers {
			wg.Add(1)
			go func(ctx context.Context, q stomp.Queue, consumerID int) {
				defer wg.Done()

				slog.Info("Starting subscriber", "queue", q.Name, "consumer", consumerID)

				for {
					select {
					case <-stopChan:
						slog.Info("Stopping subscriber", "queue", q.Name, "consumer", consumerID)
						return
					default:
						err := q.RecvAndProcessMessage(ctx)
						if err != nil {
							slog.Error("Error processing message", "queue", q.Name, "consumer", consumerID, "error", err)
						}
					}
				}
			}(ctx, q, i)
		}
	}

	<-stopChan
	slog.Info("All stomp subscribers are now running")
	wg.Wait()
	slog.Info("All stomp subscribers have stopped")
}
