package stomp

import (
	stomp "github.com/go-stomp/stomp/v3"
	"github.com/lehigh-university-libraries/scyllaridae/pkg/api"
)

func (middleware Queue) HandleIndexMessage(msg *stomp.Message, islandoraMessage *api.Payload) {

}
