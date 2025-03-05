package stomp

import (
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	stomp "github.com/go-stomp/stomp/v3"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
	"github.com/libops/isle-event-bus/internal/utils"
)

func (middleware Queue) HandleDerivativeMessage(msg *stomp.Message, islandoraMessage *api.Payload) error {
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
			return fmt.Errorf("unable to decode event message(%s): %d", err, errCode)
		}
	}

	req, err := http.NewRequest(method, middleware.Url, body)
	if err != nil {
		return fmt.Errorf("error creating HTTP request(%s): %v", middleware.Url, err)
	}

	mimeType := islandoraMessage.Attachment.Content.SourceMimeType
	if mimeType == "" && !middleware.NoPut {
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodHead, islandoraMessage.Attachment.Content.SourceURI, nil)
		if err != nil {
			return fmt.Errorf("unable to create source URI request(%s): %v", islandoraMessage.Attachment.Content.SourceURI, err)
		}

		if auth != "" {
			req.Header.Set("Authorization", auth)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("unable to get source URI (%s): %v", islandoraMessage.Attachment.Content.SourceURI, err)
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
		return fmt.Errorf("error sending HTTP GET request (%s): %v", middleware.Url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 299 {
		return fmt.Errorf("failed to deliver message (%s): %d", middleware.Url, resp.StatusCode)
	}

	if middleware.NoPut {
		return nil
	}

	putReq, err := http.NewRequest("PUT", islandoraMessage.Attachment.Content.DestinationURI, resp.Body)
	if err != nil {
		return fmt.Errorf("error creating HTTP PUT request (%s): %v", islandoraMessage.Attachment.Content.DestinationURI, err)
	}

	putReq.Header.Set("Authorization", msg.Header.Get("Authorization"))
	putReq.Header.Set("Content-Type", islandoraMessage.Attachment.Content.DestinationMimeType)
	putReq.Header.Set("Content-Location", islandoraMessage.Attachment.Content.FileUploadURI)

	// Send the PUT request
	putResp, err := client.Do(putReq)
	if err != nil {
		return fmt.Errorf("error sending HTTP PUT request (%s): %v", islandoraMessage.Attachment.Content.DestinationURI, err)
	}
	defer putResp.Body.Close()

	if putResp.StatusCode >= 299 {
		return fmt.Errorf("failed to PUT data (%s): %v", islandoraMessage.Attachment.Content.DestinationURI, err)
	}

	slog.Info("Successfully PUT data to", "url", islandoraMessage.Attachment.Content.DestinationURI, "status", putResp.StatusCode)

	return nil
}
