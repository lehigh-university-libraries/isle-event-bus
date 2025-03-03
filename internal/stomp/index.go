package stomp

import (
	"log/slog"

	stomp "github.com/go-stomp/stomp/v3"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
)

func (middleware Queue) HandleIndexMessage(msg *stomp.Message, islandoraMessage *api.Payload) {
	slog.Info("TODO", "queue", middleware.Name, "event", islandoraMessage)
}
