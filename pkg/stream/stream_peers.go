package stream

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	pkgauth "github.com/seantywork/gomehub/pkg/auth"
	"github.com/seantywork/gomehub/pkg/com"
	"github.com/seantywork/gomehub/pkg/utils"
	pkgutils "github.com/seantywork/gomehub/pkg/utils"
)

var PEERS_SIGNAL_PATH string

var PEER_SIGNAL_ATTEMPT_SYNC int

type PeersEntryStruct struct {
	RoomName []string `json:"room_name"`
}

type PeersUserStruct struct {
	UserKey string `json:"user_key"`
	User    string `json:"user"`
}

type PeersCreate struct {
	RoomName string   `json:"room_name"`
	Users    []string `json:"users"`
}

type PeersJoin struct {
	RoomName string `json:"room_name"`
	User     string `json:"user"`
	UserKey  string `json:"user_key"`
}

type ChatMessage struct {
	User    string `json:"user"`
	Message string `json:"message"`
}

var ROOMREG = make(map[string][]PeersUserStruct)

func GetPeersSignalAddress(c *gin.Context) {

	pkgauth.WhoAmI(c)

	var s_addr string

	if DEBUG {

		s_addr = INTERNAL_URL + ":" + com.CHANNEL_PORT + PEERS_SIGNAL_PATH

	} else {

		s_addr = EXTERNAL_URL + ":" + com.CHANNEL_PORT_EXTERNAL + PEERS_SIGNAL_PATH

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: s_addr})

}

func GetPeersEntry(c *gin.Context) {

	_, my_type, my_id := pkgauth.WhoAmI(c)

	pes := PeersEntryStruct{
		RoomName: make([]string, 0),
	}

	for k, v := range ROOMREG {

		if my_type == "admin" {

			pes.RoomName = append(pes.RoomName, k)

		} else {

			pu_len := len(v)

			for i := 0; i < pu_len; i++ {

				if v[i].User == my_id {

					pes.RoomName = append(pes.RoomName, k)

				}

			}

		}

	}

	pes_b, err := json.Marshal(pes)

	if err != nil {

		fmt.Printf("peers entry: marshal: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "failed to get peers entry"})

		return

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: string(pes_b)})

	return

}

func PostPeersCreate(c *gin.Context) {

	_, my_type, _ := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("peers create: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	fmt.Println("create peers")

	var req com.CLIENT_REQ

	if err := c.BindJSON(&req); err != nil {

		fmt.Printf("create peers: failed to bind: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	var p_create PeersCreate

	err := json.Unmarshal([]byte(req.Data), &p_create)

	if err != nil {

		fmt.Printf("create peers: marshal: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	ROOMREG[p_create.RoomName] = make([]PeersUserStruct, 0)

	u_len := len(p_create.Users)

	for i := 0; i < u_len; i++ {

		u_key, _ := utils.GetRandomHex(32)

		ROOMREG[p_create.RoomName] = append(ROOMREG[p_create.RoomName], PeersUserStruct{
			UserKey: u_key,
			User:    p_create.Users[i],
		})

	}

	u_key, _ := utils.GetRandomHex(32)

	ROOMREG[p_create.RoomName] = append(ROOMREG[p_create.RoomName], PeersUserStruct{
		UserKey: u_key,
		User:    "seantywork@gmail.com",
	})

	roomPeerConnections[p_create.RoomName] = []peerConnectionState{}

	roomTrackLocals[p_create.RoomName] = nil

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: fmt.Sprintf("room: %s :created", p_create.RoomName)})

	return

}

func PostPeersDelete(c *gin.Context) {

	_, my_type, _ := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("peers delete: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	fmt.Println("delete peers")

	var req com.CLIENT_REQ

	if err := c.BindJSON(&req); err != nil {

		fmt.Printf("delete peers: failed to bind: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	delete(ROOMREG, req.Data)

	delete(roomPeerConnections, req.Data)

	delete(roomTrackLocals, req.Data)

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: fmt.Sprintf("room: %s : deleted", req.Data)})

	return

}

func roomJoinAuth(c *com.ThreadSafeWriter) (string, error) {

	timeout_iter_count := 0

	timeout_iter := TIMEOUT_SEC * 10

	ticker := time.NewTicker(100 * time.Millisecond)

	received_auth := make(chan com.RT_REQ_DATA)

	got_auth := 0

	var req com.RT_REQ_DATA

	go func() {

		auth_req := com.RT_REQ_DATA{}

		err := c.ReadJSON(&auth_req)

		if err != nil {

			log.Fatal("read auth:", err)
			return
		}

		received_auth <- auth_req

	}()

	for got_auth == 0 {

		select {

		case <-ticker.C:

			if timeout_iter_count <= timeout_iter {

				timeout_iter_count += 1

			} else {

				return "", fmt.Errorf("read auth: timed out")
			}

		case a := <-received_auth:

			req = a

			got_auth = 1

			break
		}

	}

	var pj PeersJoin

	err := json.Unmarshal([]byte(req.Data), &pj)

	if err != nil {

		return "", fmt.Errorf("read auth: marshal: %s", err.Error())
	}

	p_users, okay := ROOMREG[pj.RoomName]

	if !okay {

		return "", fmt.Errorf("failed to get okay: %s", "no such room")
	}

	pu_len := len(p_users)

	found := 0

	for i := 0; i < pu_len; i++ {

		if p_users[i].User == pj.User && p_users[i].UserKey == pj.UserKey {

			found = 1

			break

		}

	}

	if found != 1 {

		return "", fmt.Errorf("no matching user found")

	}

	return pj.User, nil
}

func RoomSignalHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP request to Websocket

	UPGRADER.CheckOrigin = func(r *http.Request) bool { return true }

	unsafeConn, err := UPGRADER.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade: %s\n", err.Error())
		return
	}

	roomParam := strings.TrimPrefix(r.URL.Path, PEERS_SIGNAL_PATH)

	log.Printf("room: %s\n", roomParam)

	_, okay := roomPeerConnections[roomParam]

	if !okay {

		log.Printf("no such room: %s\n", roomParam)

		return
	}

	c := &com.ThreadSafeWriter{unsafeConn, sync.Mutex{}}

	// When this frame returns close the Websocket
	defer c.Close() //nolint

	thisUser, err := roomJoinAuth(c)

	if err != nil {

		log.Print("auth:", err)

		return
	}
	log.Printf("auth success: user: %s, room: %s\n", thisUser, roomParam)

	// Create new PeerConnection
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs:       []string{TURN_SERVER_ADDR[0].Addr},
				Username:   TURN_SERVER_ADDR[0].Id,
				Credential: TURN_SERVER_ADDR[0].Pw,
			},
		},
	})

	if err != nil {
		log.Print(err)
		return
	}

	log.Print("new peerconnection added")

	// When this frame returns close the PeerConnection
	defer peerConnection.Close() //nolint

	// Accept one audio and one video track incoming
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Print(err)
			return
		}
	}

	// Add our new PeerConnection to global list
	com.ListLock.Lock()
	roomPeerConnections[roomParam] = append(roomPeerConnections[roomParam], peerConnectionState{peerConnection, c})
	com.ListLock.Unlock()

	// Trickle ICE. Emit server candidate to client
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {

		log.Printf("got ice candidate\n")

		if i == nil {
			return
		}

		candidateStringEnc := pkgutils.Encode(i.ToJSON())

		if writeErr := c.WriteJSON(&SIGNAL_INFO{
			Command: "candidate",
			Data:    candidateStringEnc,
		}); writeErr != nil {
			log.Println(writeErr)
		}

		log.Printf("sent ice candidate\n")
	})

	// If PeerConnection is closed remove it from global list
	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			log.Printf("on connection state change: %s \n", p.String())
			if err := peerConnection.Close(); err != nil {
				log.Print(err)
			}
		case webrtc.PeerConnectionStateClosed:
			log.Printf("on connection state change: %s \n", p.String())
			signalPeerConnections(roomParam)
		default:
			log.Printf("on connection state change: %s \n", p.String())
		}
	})

	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		// Create a track to fan out our incoming video to all peers
		trackLocal := addTrack(roomParam, t)
		defer removeTrack(roomParam, trackLocal)

		buf := make([]byte, 1500)
		for {
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}

			if _, err = trackLocal.Write(buf[:i]); err != nil {
				return
			}
		}
	})

	// Signal for the new PeerConnection

	signalPeerConnections(roomParam)

	message := &SIGNAL_INFO{}
	for {

		_, raw, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		} else if err := json.Unmarshal(raw, &message); err != nil {
			log.Println(err)
			return
		}

		log.Printf("got message: %s\n", message.Command)

		switch message.Command {
		case "candidate":

			log.Printf("got client ice candidate")

			candidate := webrtc.ICECandidateInit{}

			pkgutils.Decode(message.Data, &candidate)

			/*
				if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
					log.Println(err)
					return
				}

			*/
			if err := peerConnection.AddICECandidate(candidate); err != nil {
				log.Println(err)
				return
			}

			log.Printf("added client ice candidiate")

		case "answer":
			answer := webrtc.SessionDescription{}

			pkgutils.Decode(message.Data, &answer)

			/*
				if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
					log.Println(err)
					return
				}
			*/

			if err := peerConnection.SetRemoteDescription(answer); err != nil {
				log.Println(err)
				return
			}

		case "chat":

			cm := ChatMessage{
				Message: html.EscapeString(message.Data),
				User:    thisUser,
			}

			jb, err := json.Marshal(cm)

			if err != nil {

				log.Println(err)

				return
			}

			message.Data = string(jb)

			broadcastPeerConnections(roomParam, message)

		}
	}
}

func dispatchKeyFrame(k string) {
	com.ListLock.Lock()
	defer com.ListLock.Unlock()

	for i := range roomPeerConnections[k] {
		for _, receiver := range roomPeerConnections[k][i].peerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = roomPeerConnections[k][i].peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}

}

func broadcastPeerConnections(roomName string, message *SIGNAL_INFO) {

	for i := range roomPeerConnections[roomName] {

		roomPeerConnections[roomName][i].websocket.WriteJSON(*message)

	}

}

func signalPeerConnections(k string) {
	com.ListLock.Lock()

	defer func() {
		com.ListLock.Unlock()
		dispatchKeyFrame(k)
	}()

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == PEER_SIGNAL_ATTEMPT_SYNC {
			// We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				signalPeerConnections(k)
			}()
			return
		}

		if !attemptSync(k) {

			break
		}
	}
}

func attemptSync(k string) bool {

	for i := range roomPeerConnections[k] {
		if roomPeerConnections[k][i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
			roomPeerConnections[k] = append(roomPeerConnections[k][:i], roomPeerConnections[k][i+1:]...)
			return true // We modified the slice, start from the beginning
		}

		// map of sender we already are sending, so we don't double send
		existingSenders := map[string]bool{}

		for _, sender := range roomPeerConnections[k][i].peerConnection.GetSenders() {
			if sender.Track() == nil {
				continue
			}

			existingSenders[sender.Track().ID()] = true

			// If we have a RTPSender that doesn't map to a existing track remove and signal
			if _, ok := roomTrackLocals[k][sender.Track().ID()]; !ok {

				if err := roomPeerConnections[k][i].peerConnection.RemoveTrack(sender); err != nil {
					return true
				}
			}
		}

		// Don't receive videos we are sending, make sure we don't have loopback

		for _, receiver := range roomPeerConnections[k][i].peerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			existingSenders[receiver.Track().ID()] = true

		}

		// Add all track we aren't sending yet to the PeerConnection
		for trackID, t := range roomTrackLocals[k] {

			if _, ok := existingSenders[trackID]; !ok {

				if _, err := roomPeerConnections[k][i].peerConnection.AddTrack(t); err != nil {
					return true
				}
			}
		}

		offer, err := roomPeerConnections[k][i].peerConnection.CreateOffer(nil)
		if err != nil {
			return true
		}

		if err = roomPeerConnections[k][i].peerConnection.SetLocalDescription(offer); err != nil {
			return true
		}

		offerStringEnc := pkgutils.Encode(offer)

		if err = roomPeerConnections[k][i].websocket.WriteJSON(&SIGNAL_INFO{
			Command: "offer",
			Data:    offerStringEnc,
		}); err != nil {
			return true
		}
	}

	return false
}

func RoomSignalHandlerSingle(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP request to Websocket

	UPGRADER.CheckOrigin = func(r *http.Request) bool { return true }

	unsafeConn, err := UPGRADER.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade: %s\b", err.Error())
		return
	}

	roomParam := strings.TrimPrefix(r.URL.Path, PEERS_SIGNAL_PATH)

	log.Printf("single room: %s\n", roomParam)

	c := &com.ThreadSafeWriter{unsafeConn, sync.Mutex{}}

	// When this frame returns close the Websocket
	defer c.Close() //nolint

	thisUser, err := roomJoinAuth(c)

	if err != nil {

		log.Print("auth:", err)

		return
	}
	log.Printf("auth success: user: %s, room: %s\n", thisUser, roomParam)

	// Create new PeerConnection
	peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs:       []string{TURN_SERVER_ADDR[0].Addr},
				Username:   TURN_SERVER_ADDR[0].Id,
				Credential: TURN_SERVER_ADDR[0].Pw,
			},
		},
	})

	if err != nil {
		log.Print(err)
		return
	}

	log.Print("new peerconnection added")

	// When this frame returns close the PeerConnection
	defer peerConnection.Close() //nolint

	// Accept one audio and one video track incoming
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err := peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Print(err)
			return
		}
	}

	// Add our new PeerConnection to global list
	com.ListLock.Lock()
	roomPeerConnectionsSingle = append(roomPeerConnectionsSingle, peerConnectionState{peerConnection, c})
	com.ListLock.Unlock()

	// Trickle ICE. Emit server candidate to client
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {

		log.Printf("got ice candidate\n")

		if i == nil {
			return
		}

		candidateStringEnc := pkgutils.Encode(i.ToJSON())

		if writeErr := c.WriteJSON(&SIGNAL_INFO{
			Command: "candidate",
			Data:    candidateStringEnc,
		}); writeErr != nil {
			log.Println(writeErr)
		}

		log.Printf("sent ice candidate\n")
	})

	// If PeerConnection is closed remove it from global list
	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			log.Printf("on connection state change: %s \n", p.String())
			if err := peerConnection.Close(); err != nil {
				log.Print(err)
			}
		case webrtc.PeerConnectionStateClosed:
			log.Printf("on connection state change: %s \n", p.String())
			signalPeerConnectionsSingle()
		default:
			log.Printf("on connection state change: %s \n", p.String())
		}
	})

	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		// Create a track to fan out our incoming video to all peers
		trackLocal := addTrackSingle(t)
		defer removeTrackSingle(trackLocal)

		buf := make([]byte, 1500)
		for {
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}

			if _, err = trackLocal.Write(buf[:i]); err != nil {
				return
			}
		}
	})

	// Signal for the new PeerConnection

	signalPeerConnectionsSingle()

	message := &SIGNAL_INFO{}
	for {

		_, raw, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		} else if err := json.Unmarshal(raw, &message); err != nil {
			log.Println(err)
			return
		}

		log.Printf("got message: %s\n", message.Command)

		switch message.Command {
		case "candidate":

			log.Printf("got client ice candidate")

			candidate := webrtc.ICECandidateInit{}

			pkgutils.Decode(message.Data, &candidate)

			/*
				if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
					log.Println(err)
					return
				}

			*/
			if err := peerConnection.AddICECandidate(candidate); err != nil {
				log.Println(err)
				return
			}

			log.Printf("added client ice candidiate")

		case "answer":
			answer := webrtc.SessionDescription{}

			pkgutils.Decode(message.Data, &answer)

			/*
				if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
					log.Println(err)
					return
				}
			*/

			if err := peerConnection.SetRemoteDescription(answer); err != nil {
				log.Println(err)
				return
			}

		case "chat":

			cm := ChatMessage{
				Message: html.EscapeString(message.Data),
				User:    thisUser,
			}

			jb, err := json.Marshal(cm)

			if err != nil {

				log.Println(err)

				return
			}

			message.Data = string(jb)

			broadcastPeerConnectionsSingle(message)

		}
	}
}

func dispatchKeyFrameSingle() {
	com.ListLock.Lock()
	defer com.ListLock.Unlock()

	for i := range roomPeerConnectionsSingle {
		for _, receiver := range roomPeerConnectionsSingle[i].peerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = roomPeerConnectionsSingle[i].peerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}

}

func broadcastPeerConnectionsSingle(message *SIGNAL_INFO) {

	for i := range roomPeerConnectionsSingle {

		roomPeerConnectionsSingle[i].websocket.WriteJSON(*message)

	}

}

func signalPeerConnectionsSingle() {
	com.ListLock.Lock()

	defer func() {
		com.ListLock.Unlock()
		dispatchKeyFrameSingle()
	}()

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == PEER_SIGNAL_ATTEMPT_SYNC {
			// We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				signalPeerConnectionsSingle()
			}()
			return
		}

		if !attemptSyncSingle() {

			break
		}
	}
}

func attemptSyncSingle() bool {

	for i := range roomPeerConnectionsSingle {
		if roomPeerConnectionsSingle[i].peerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
			roomPeerConnectionsSingle = append(roomPeerConnectionsSingle[:i], roomPeerConnectionsSingle[i+1:]...)
			return true // We modified the slice, start from the beginning
		}

		// map of sender we already are seanding, so we don't double send
		existingSenders := map[string]bool{}

		for _, sender := range roomPeerConnectionsSingle[i].peerConnection.GetSenders() {
			if sender.Track() == nil {
				continue
			}

			existingSenders[sender.Track().ID()] = true

			// If we have a RTPSender that doesn't map to a existing track remove and signal
			if _, ok := roomTrackLocalsSingle[sender.Track().ID()]; !ok {

				if err := roomPeerConnectionsSingle[i].peerConnection.RemoveTrack(sender); err != nil {
					return true
				}
			}
		}

		// Don't receive videos we are sending, make sure we don't have loopback

		for _, receiver := range roomPeerConnectionsSingle[i].peerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			existingSenders[receiver.Track().ID()] = true

		}

		// Add all track we aren't sending yet to the PeerConnection
		for trackID, t := range roomTrackLocalsSingle {

			if _, ok := existingSenders[trackID]; !ok {

				if _, err := roomPeerConnectionsSingle[i].peerConnection.AddTrack(t); err != nil {
					return true
				}
			}
		}

		offer, err := roomPeerConnectionsSingle[i].peerConnection.CreateOffer(nil)
		if err != nil {
			return true
		}

		if err = roomPeerConnectionsSingle[i].peerConnection.SetLocalDescription(offer); err != nil {
			return true
		}

		offerStringEnc := pkgutils.Encode(offer)

		if err = roomPeerConnectionsSingle[i].websocket.WriteJSON(&SIGNAL_INFO{
			Command: "offer",
			Data:    offerStringEnc,
		}); err != nil {
			return true
		}
	}

	return false
}
