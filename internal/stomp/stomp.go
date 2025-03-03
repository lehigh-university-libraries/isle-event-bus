package stomp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	stomp "github.com/go-stomp/stomp/v3"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
)

type Queue struct {
	Name        string `yaml:"queueName"`
	Url         string `yaml:"url"`
	Type        string `yaml:"type"`
	Consumers   int    `yaml:"consumers"`
	ForwardAuth bool   `yaml:"forwardAuth,omitempty"`
	// for services that do not require creating a derivative, set to true
	NoPut bool `yaml:"noPut,omitempty"`
	// whether or not to directly upload the file supplied in the islandora event
	// to the scyllaridae service
	PutFile bool `yaml:"putFile,omitempty"`
}

func (middleware Queue) HandleMessage(msg *stomp.Message) {
	islandoraMessage, err := api.DecodeEventMessage(msg.Body)
	if err != nil {
		slog.Error("Unable to decode event message", "err", err)
		return
	}

	if middleware.Type == "index" {
		middleware.HandleIndexMessage(msg, &islandoraMessage)
		return
	}
	middleware.HandleDerivativeMessage(msg, &islandoraMessage)
}

func (middleware Queue) RecvAndProcessMessage(ctx context.Context) error {
	addr := os.Getenv("STOMP_SERVER_ADDR")
	if addr == "" {
		addr = "activemq:61613"
	}

	c, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Error("Cannot connect to port", "queue", middleware.Name, "err", err.Error())
		return err
	}
	tcpConn := c.(*net.TCPConn)

	err = tcpConn.SetKeepAlive(true)
	if err != nil {
		slog.Error("Cannot set keepalive", "queue", middleware.Name, "err", err.Error())
		return err
	}

	err = tcpConn.SetKeepAlivePeriod(10 * time.Second)
	if err != nil {
		slog.Error("Cannot set keepalive period", "queue", middleware.Name, "err", err.Error())
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			for {
				conn, err := stomp.Connect(tcpConn,
					stomp.ConnOpt.HeartBeat(10*time.Second, 10*time.Second),
					stomp.ConnOpt.HeartBeatGracePeriodMultiplier(1.5),
					stomp.ConnOpt.HeartBeatError(60*time.Second),
				)
				if err != nil {
					slog.Error("Cannot connect to STOMP server", "queue", middleware.Name, "err", err.Error())
					return err
				}
				defer func() {
					err := conn.Disconnect()
					if err != nil {
						slog.Error("Problem disconnecting from STOMP server", "err", err)
					}
				}()
				sub, err := conn.Subscribe(middleware.Name, stomp.AckClient)
				if err != nil {
					slog.Error("Cannot subscribe to queue", "queue", middleware.Name, "err", err.Error())
					return err
				}
				defer func() {
					if !sub.Active() {
						return
					}
					err := sub.Unsubscribe()
					if err != nil {
						slog.Error("Problem unsubscribing", "err", err)
					}
				}()
				slog.Info("Subscribed to queue", "queue", middleware.Name)

				for {
					// Wait for the next message (blocks if the channel is empty)
					msg, ok := <-sub.C
					if !ok {
						return fmt.Errorf("subscription to %s is closed", middleware.Name)
					}

					if msg == nil || len(msg.Body) == 0 {
						if !sub.Active() {
							return fmt.Errorf("no longer subscribed to %s", middleware.Name)
						}
						continue
					}

					// Process the message synchronously
					middleware.HandleMessage(msg)

					err := msg.Conn.Ack(msg)
					if err != nil {
						slog.Error("Failed to acknowledge message", "queue", middleware.Name, "error", err)
					}
				}
			}
		}
	}
}
