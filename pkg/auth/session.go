package auth

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/seantywork/gomehub/pkg/dbquery"
	pkglog "github.com/seantywork/gomehub/pkg/log"
)

func WhoAmI(c *gin.Context) (string, string, string) {

	var your_key string = ""

	var your_type string = ""

	var your_id string = ""

	session := sessions.Default(c)

	var session_id string

	route_key := c.Request.URL.Path

	header, err := json.Marshal(c.Request.Header)

	var header_string string

	if err != nil {

		header_string = err.Error()

	} else {

		header_string = string(header)

	}

	pkglog.PushLog(route_key, header_string)

	v := session.Get("SOLIAGAIN")

	if v == nil {

		fmt.Printf("you: %s\n", "nobody-yet")
		return "", "", ""

	} else {

		session_id = v.(string)

	}

	ss, err := dbquery.GetBySessionKeyFromSession(session_id)

	if err != nil {

		fmt.Printf("you: %s\n", "nodody-no-session")

		return "", "", ""
	}

	your_key = session_id
	your_type = ss.Type
	your_id = ss.Id

	return your_key, your_type, your_id
}
