package stomp

import (
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"

	stomp "github.com/go-stomp/stomp/v3"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
	"github.com/libops/riq/internal/utils"
)

func (middleware Queue) HandleDerivativeMessage(msg *stomp.Message, islandoraMessage *api.Payload) {
	var (
		body    io.ReadCloser
		errCode int
		err     error
		method  = http.MethodGet
		auth    = msg.Header.Get("Authorization")
	)
	if middleware.PutFile {
		method = http.MethodPost
		body, errCode, err = utils.GetFileStream(islandoraMessage, auth)
		if err != nil {
			slog.Error("Unable to decode event message", "err", err, "code", errCode)
			return
		}
	}

	req, err := http.NewRequest(method, middleware.Url, body)
	if err != nil {
		slog.Error("Error creating HTTP request", "url", middleware.Url, "err", err)
		return
	}

	mimeType := islandoraMessage.Attachment.Content.SourceMimeType
	if mimeType == "" && !middleware.NoPut {
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
