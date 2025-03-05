package stomp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	stomp "github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestHandleIndexMessage(t *testing.T) {
	// Mock JSON-LD Server
	jsonldData := `{"@context": "https://schema.org", "@type": "CreativeWork", "name": "Test Object"}`
	jsonldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/ld+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jsonldData))
	}))
	defer jsonldServer.Close()

	// Mock Receiving Server
	var receivedRequests []string
	var mu sync.Mutex
	receivingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		body, _ := io.ReadAll(r.Body)
		receivedRequests = append(receivedRequests, string(body))

		w.WriteHeader(http.StatusOK)
	}))
	defer receivingServer.Close()

	// Construct the Queue with the mock URL
	mockQueue := Queue{
		Url:              receivingServer.URL + "/:uuid",
		EventMethod:      http.MethodPost,
		LocationMimetype: "application/ld+json",
	}

	// Construct the test event
	event := &api.Payload{
		Object: api.Object{
			ID: "urn:uuid:1234",
			URL: []api.Link{
				{
					MediaType: "application/ld+json",
					Href:      jsonldServer.URL,
				},
			},
			IsNewVersion: true,
		},
		Target: "fedora",
	}

	// Mock stomp message
	msg := &stomp.Message{
		Header: frame.NewHeader("Authorization", "Bearer test-token"),
	}

	// Call the function
	err := mockQueue.HandleIndexMessage(msg, event)
	assert.Nil(t, err)

	// Assertions
	assert.Len(t, receivedRequests, 2, "Expected two requests to the receiving server")
}
