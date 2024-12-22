package stream

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	ch "github.com/seantywork/gomehub/pkg/com"
)

type SIGNAL_INFO struct {
	Command string `json:"command"`
	Status  string `json:"status"`
	Data    string `json:"data"`
}

var UPGRADER = websocket.Upgrader{}

var roomPeerConnections = make(map[string][]peerConnectionState)

var roomTrackLocals = make(map[string]map[string]*webrtc.TrackLocalStaticRTP)

var roomPeerConnectionsSingle = make([]peerConnectionState, 0)

var roomTrackLocalsSingle = make(map[string]*webrtc.TrackLocalStaticRTP)

type peerConnectionState struct {
	peerConnection *webrtc.PeerConnection
	websocket      *ch.ThreadSafeWriter
}

func SignalDispatcher() {

	for range time.NewTicker(time.Second * RTCP_PLI_INTERVAL).C {

		for k, _ := range roomPeerConnections {

			dispatchKeyFrame(k)
		}

	}
}

func SignalDispatcherSingle() {

	for range time.NewTicker(time.Second * RTCP_PLI_INTERVAL).C {

		dispatchKeyFrameSingle()

	}
}
