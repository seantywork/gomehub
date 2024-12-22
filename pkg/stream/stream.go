package stream

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pion/ice/v3"
	"github.com/pion/webrtc/v4"
	"github.com/seantywork/gomehub/pkg/com"
)

var DEBUG bool

var EXTERNAL_URL string

var INTERNAL_URL string

var RTCP_PLI_INTERVAL time.Duration

var UDP_MUX_PORT int

var UDP_EPHEMERAL_PORT_MIN int

var UDP_EPHEMERAL_PORT_MAX int

var TIMEOUT_SEC int

var SINGLE_ROOM_MODE bool

var TURN_SERVER_ADDR []struct {
	Addr string `json:"addr"`
	Id   string `json:"id"`
	Pw   string `json:"pw"`
}

var api *webrtc.API

func GetTurnServeAddr(c *gin.Context) {

	data_b, err := json.Marshal(TURN_SERVER_ADDR)

	if err != nil {

		c.JSON(http.StatusOK, com.SERVER_RE{Status: "failed", Reply: "error"})
	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: string(data_b)})

}

func InitWebRTCApi() {

	settingEngine := webrtc.SettingEngine{}

	var filterFunc func(string) bool = func(ifname string) bool {

		if strings.HasPrefix(ifname, "br-") {

			return false
		} else if strings.HasPrefix(ifname, "vir") {

			return false
		} else if strings.HasPrefix(ifname, "docker") {

			return false
		}

		return true

	}

	ifaceFilter := ice.UDPMuxFromPortWithInterfaceFilter(filterFunc)
	mux, err := ice.NewMultiUDPMuxFromPort(UDP_MUX_PORT, ifaceFilter)

	log.Println("creating webrtc api")

	settingEngine.SetICEUDPMux(mux)
	if err != nil {
		panic(err)
	}

	log.Println("created webrtc api")

	settingEngine.SetEphemeralUDPPortRange(uint16(UDP_EPHEMERAL_PORT_MIN), uint16(UDP_EPHEMERAL_PORT_MAX))

	api = webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))

}

func addTrack(k string, t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	com.ListLock.Lock()
	defer func() {
		com.ListLock.Unlock()
		signalPeerConnections(k)
	}()

	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	if roomTrackLocals[k] == nil {
		roomTrackLocals[k] = make(map[string]*webrtc.TrackLocalStaticRTP)
	}

	roomTrackLocals[k][t.ID()] = trackLocal
	return trackLocal
}

func removeTrack(k string, t *webrtc.TrackLocalStaticRTP) {
	com.ListLock.Lock()
	defer func() {
		com.ListLock.Unlock()
		signalPeerConnections(k)
	}()

	delete(roomTrackLocals, k)
}

func addTrackSingle(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	com.ListLock.Lock()
	defer func() {
		com.ListLock.Unlock()
		signalPeerConnectionsSingle()
	}()

	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	roomTrackLocalsSingle[t.ID()] = trackLocal
	return trackLocal
}

func removeTrackSingle(t *webrtc.TrackLocalStaticRTP) {
	com.ListLock.Lock()
	defer func() {
		com.ListLock.Unlock()
		signalPeerConnectionsSingle()
	}()

	delete(roomTrackLocalsSingle, t.ID())
}
