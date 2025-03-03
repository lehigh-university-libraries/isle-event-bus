package utils

import (
	"fmt"
	"io"
	"net/http"

	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
)

func GetFileStream(message *api.Payload, auth string) (io.ReadCloser, int, error) {
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

	message.Attachment.Content.SourceMimeType = sourceResp.Header.Get("Content-Type")
	return sourceResp.Body, http.StatusOK, nil
}
