package sorrylinus

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	pkgauth "github.com/seantywork/gomehub/pkg/auth"
	"github.com/seantywork/gomehub/pkg/com"
	pkgutils "github.com/seantywork/gomehub/pkg/utils"
)

var DEBUG bool

var SOLI_FRONT_ADDR string

var SOLI_SIGNAL_PATH string

var INTERNAL_URL string

var EXTERNAL_URL string

var TIMEOUT_SEC int

var UREG = make(map[string]string)

var SOLIREG = make(map[string]*websocket.Conn)

var UPGRADER = websocket.Upgrader{}

type SoliCreate struct {
	User       string `json:"user"`
	Passphrase string `json:"passphrase"`
}

type SoliInfo struct {
	User string `json:"user"`
	Key  string `json:"key"`
}

func GetSoliSignalAddress(c *gin.Context) {

	var s_addr string

	if DEBUG {

		s_addr = INTERNAL_URL + ":" + com.CHANNEL_PORT + SOLI_SIGNAL_PATH

	} else {

		s_addr = EXTERNAL_URL + ":" + com.CHANNEL_PORT_EXTERNAL + SOLI_SIGNAL_PATH

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: s_addr})

}

func PostSoliOpen(c *gin.Context) {

	_, my_type, my_id := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("soli open: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	var req com.CLIENT_REQ

	if err := c.BindJSON(&req); err != nil {

		fmt.Printf("soli open: bind: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	var sc SoliCreate

	err := json.Unmarshal([]byte(req.Data), &sc)

	if err != nil {

		fmt.Printf("soli open: bind: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	_, okay := UREG[my_id]

	if okay {

		fmt.Printf("soli open: already opened: %s\n", my_id)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "already opened"})

		return

	}

	soli_c, err := connectToSorrylinus(sc.User, sc.Passphrase)

	if err != nil {

		fmt.Printf("soli open: failed to connect: %s\n", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to connect to sorrylinus"})

		return

	}

	log.Printf("connected to sorrylinus\n")

	token, _ := pkgutils.GetRandomHex(32)

	UREG[my_id] = token

	SOLIREG[my_id] = soli_c

	si := SoliInfo{
		User: my_id,
		Key:  token,
	}

	jb, err := json.Marshal(si)

	if err != nil {

		fmt.Printf("soli open: marshal: %s\n", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to open"})

		return

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: string(jb)})

	return

}

func PostSoliClose(c *gin.Context) {

	_, my_type, my_id := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("soli open: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	delete(UREG, my_id)

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: fmt.Sprintf("soli: %s: closed", my_id)})

	return
}

func SoliSignalHandler(w http.ResponseWriter, r *http.Request) {

	UPGRADER.CheckOrigin = func(r *http.Request) bool { return true }

	c, err := UPGRADER.Upgrade(w, r, nil)

	if err != nil {
		log.Printf("upgrade: %s\n", err.Error())
		return
	}

	defer c.Close()

	c_ret := com.RT_RESP_DATA{}

	log.Printf("entered soli signal handler\n")

	thisUser, err := soliEnterAuth(c)

	if err != nil {

		fmt.Printf("soli enter auth: %s\n", err.Error())

		c_ret.Status = "error"

		c_ret.Data = "failed to auth"

		c.WriteJSON(c_ret)

		return

	}

	log.Printf("user: %s\n", thisUser)

	c_ret.Status = "success"

	c_ret.Data = "success auth"

	c.WriteJSON(c_ret)

	for {

		var resp *com.RT_RESP_DATA
		req := com.RT_REQ_DATA{}

		err := c.ReadJSON(&req)

		if err != nil {

			log.Printf("rt: read: %s", err.Error())

			return

		}

		resp, err = RoundTrip(thisUser, &req)

		if err != nil {

			c_ret.Status = "error"

			c_ret.Data = err.Error()

			c.WriteJSON(c_ret)

			log.Printf("rt: error: %s", err.Error())

			return
		}

		c.WriteJSON(resp)

	}

}

func soliEnterAuth(c *websocket.Conn) (string, error) {

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

			log.Printf("read auth: %s\n", err.Error())
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

				return "", fmt.Errorf("read auth timed out")
			}

		case a := <-received_auth:

			req = a

			got_auth = 1

			break
		}

	}

	var si SoliInfo

	err := json.Unmarshal([]byte(req.Data), &si)

	if err != nil {

		return "", err
	}

	token, okay := UREG[si.User]

	if !okay {

		return "", fmt.Errorf("no such user: %s", si.User)

	}

	if si.Key != token {

		return "", fmt.Errorf("key not matching for user: %s", si.User)

	}

	return si.User, nil
}

func connectToSorrylinus(user string, passphrase string) (*websocket.Conn, error) {

	log.Printf("connecting soli front: %s\n", SOLI_FRONT_ADDR)

	u := SOLI_FRONT_ADDR

	c, _, err := websocket.DefaultDialer.Dial(u, nil)

	if err != nil {
		return nil, fmt.Errorf("dial:", err.Error())
	}

	var req com.RT_REQ_DATA

	req.Command = "auth"
	req.Data = user + ":" + passphrase

	err = c.WriteJSON(req)

	if err != nil {
		return nil, fmt.Errorf("write: %s", err.Error())
	}

	var resp com.RT_RESP_DATA

	err = c.ReadJSON(&resp)

	if err != nil {

		return nil, fmt.Errorf("read: %s", err.Error())

	}

	if resp.Status != "success" {

		return nil, fmt.Errorf("auth: failed: %s", resp.Data)
	}

	return c, nil
}
