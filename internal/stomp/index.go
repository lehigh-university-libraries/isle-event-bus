package stomp

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	stomp "github.com/go-stomp/stomp/v3"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
)

func (q Queue) HandleIndexMessage(msg *stomp.Message, event *api.Payload) error {
	uuid := strings.Replace(event.Object.ID, "urn:uuid:", "", 1)
	slog.Info("Got UUID", "uuid", uuid)
	jsonldUrl := ""
	for _, url := range event.Object.URL {
		if url.MediaType == "application/ld+json" {
			jsonldUrl = url.Href
			break
		}
	}

	if jsonldUrl == "" {
		return fmt.Errorf("can not process message with no JSON LD URL: %v", event)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errChan := make(chan error, 1)

	url := strings.Replace(q.Url, ":uuid", uuid, 1)
	url = strings.Replace(url, ":sourceField", event.Attachment.Content.SourceField, 1)
	versionUrl := url + "/version"
	auth := msg.Header.Get("Authorization")

	go func() {
		defer wg.Done()
		if err := q.sendRequest(http.MethodPost, url, auth, jsonldUrl, event); err != nil {
			errChan <- err

		}
	}()

	go func() {
		defer wg.Done()
		if q.EventMethod != http.MethodDelete && event.Object.IsNewVersion {
			if err := q.sendRequest(http.MethodPost, versionUrl, auth, jsonldUrl, event); err != nil {
				errChan <- err
			}
		}
	}()

	wg.Wait()

	close(errChan)
	for err := range errChan {
		return fmt.Errorf("error sending index event: %v", err)
	}

	slog.Info("Processed event", "uuid", event.Object.ID)

	return nil
}

func (q Queue) sendRequest(method, url, auth, contentLocation string, event *api.Payload) error {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request (%s): %v", url, err)
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("X-Islandora-Fedora-Endpoint", event.Target)

	if contentLocation != "" {
		req.Header.Set("Content-Location", contentLocation)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Request failed", "error", err)
		return fmt.Errorf("request failed (%s): %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("server error (%s): %d", url, resp.StatusCode)
	}

	slog.Info("Request sent", "url", url, "status", resp.StatusCode)

	return nil
}
