package config

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	stomp "github.com/go-stomp/stomp/v3"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
	yaml "gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Queues []Queue `yaml:"queues,omitempty"`
}

type Queue struct {
	Name        string `yaml:"queueName"`
	Url         string `yaml:"url"`
	Consumers   int    `yaml:"consumers,omitempty"`
	ForwardAuth bool   `yaml:"forwardAuth,omitempty"`
	// for services that do not require creating a derivative, set to true
	NoPut bool `yaml:"noPut"`
	// whether or not to directly upload the file supplied in the islandora event
	// to the scyllaridae service
	PutFile bool `yaml:"putFile"`
}

func ReadConfig(yp string) (*ServerConfig, error) {
	var (
		y   []byte
		err error
	)
	yml := os.Getenv("RIQ_YML")
	if yml != "" {
		y = []byte(yml)
	} else {
		y, err = os.ReadFile(yp)
		if err != nil {
			return nil, err
		}
	}

	var c ServerConfig
	err = yaml.Unmarshal(y, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func GetFileStream(message api.Payload, auth string) (io.ReadCloser, int, error) {
	req, err := http.NewRequest("GET", message.Attachment.Content.SourceURI, nil)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("bad request")
	}
	req.Header.Set("Authorization", auth)
	sourceResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("internal error")
	}
	if sourceResp.StatusCode != http.StatusOK {
		return nil, http.StatusFailedDependency, fmt.Errorf("failed dependency")
	}

	return sourceResp.Body, http.StatusOK, nil
}

func (middleware Queue) HandleMessage(msg *stomp.Message) {
	islandoraMessage, err := api.DecodeEventMessage(msg.Body)
	if err != nil {
		slog.Error("Unable to decode event message", "err", err)
		return
	}

	method := http.MethodGet
	var (
		body    io.ReadCloser
		errCode int
	)
	auth := msg.Header.Get("Authorization")
	if middleware.PutFile {
		method = http.MethodPost
		body, errCode, err = GetFileStream(islandoraMessage, auth)
		if err != nil {
			slog.Error("Unable to decode event message", "err", err, "code", errCode)
			return
		}
	} else {
		body = nil
	}

	req, err := http.NewRequest(method, middleware.Url, body)
	if err != nil {
		slog.Error("Error creating HTTP request", "url", middleware.Url, "err", err)
		return
	}

	mimeType := islandoraMessage.Attachment.Content.SourceMimeType
	if mimeType == "" {
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodHead, islandoraMessage.Attachment.Content.SourceURI, nil)
		if err != nil {
			slog.Error("Unable to create source URI request", "uri", islandoraMessage.Attachment.Content.SourceURI, "err", err)
			return
		}

		if auth != "" {
			req.Header.Set("Authorization", auth)
		}

		resp, err := client.Do(req)
		if err != nil {
			slog.Error("Unable to get source URI", "uri", islandoraMessage.Attachment.Content.SourceURI, "err", err)
			return
		}
		defer resp.Body.Close()
		mimeType = resp.Header.Get("Content-Type")

	}
	req.Header.Set("X-Islandora-Event", base64.StdEncoding.EncodeToString(msg.Body))
	req.Header.Set("Accept", islandoraMessage.Attachment.Content.DestinationMimeType)
	req.Header.Set("Content-Type", mimeType)

	if middleware.ForwardAuth {
		auth := msg.Header.Get("Authorization")
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending HTTP GET request", "url", middleware.Url, "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 299 {
		slog.Error("Failed to deliver message", "url", middleware.Url, "status", resp.StatusCode)
		return
	}

	if middleware.NoPut {
		return
	}

	putReq, err := http.NewRequest("PUT", islandoraMessage.Attachment.Content.DestinationURI, resp.Body)
	if err != nil {
		slog.Error("Error creating HTTP PUT request", "url", islandoraMessage.Attachment.Content.DestinationURI, "err", err)
		return
	}

	putReq.Header.Set("Authorization", msg.Header.Get("Authorization"))
	putReq.Header.Set("Content-Type", islandoraMessage.Attachment.Content.DestinationMimeType)
	putReq.Header.Set("Content-Location", islandoraMessage.Attachment.Content.FileUploadURI)

	// Send the PUT request
	putResp, err := client.Do(putReq)
	if err != nil {
		slog.Error("Error sending HTTP PUT request", "url", islandoraMessage.Attachment.Content.DestinationURI, "err", err)
		return
	}
	defer putResp.Body.Close()

	if putResp.StatusCode >= 299 {
		slog.Error("Failed to PUT data", "url", islandoraMessage.Attachment.Content.DestinationURI, "status", putResp.StatusCode)
	} else {
		slog.Info("Successfully PUT data to", "url", islandoraMessage.Attachment.Content.DestinationURI, "status", putResp.StatusCode)
	}
}

func (middleware Queue) RecvAndProcessMessage() error {
	addr := os.Getenv("STOMP_SERVER_ADDR")
	if addr == "" {
		addr = "activemq:61613"
	}

	c, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Error("Cannot connect to port", "err", err.Error())
		return err
	}
	tcpConn := c.(*net.TCPConn)

	err = tcpConn.SetKeepAlive(true)
	if err != nil {
		slog.Error("Cannot set keepalive", "err", err.Error())
		return err
	}

	err = tcpConn.SetKeepAlivePeriod(10 * time.Second)
	if err != nil {
		slog.Error("Cannot set keepalive period", "err", err.Error())
		return err
	}
	for {
		conn, err := stomp.Connect(tcpConn,
			stomp.ConnOpt.HeartBeat(10*time.Second, 10*time.Second),
			stomp.ConnOpt.HeartBeatGracePeriodMultiplier(1.5),
			stomp.ConnOpt.HeartBeatError(60*time.Second),
		)
		if err != nil {
			slog.Error("Cannot connect to STOMP server", "err", err.Error())
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
