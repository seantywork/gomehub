package manage

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	pkgauth "github.com/seantywork/gomehub/pkg/auth"
	"github.com/seantywork/gomehub/pkg/com"
	pkglog "github.com/seantywork/gomehub/pkg/log"
)

func GetManualLogFlush(c *gin.Context) {

	log.Println("incoming flush log")

	_, my_type, _ := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("log flush: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}
	var resp com.SERVER_RE

	pkglog.LogFlush()

	resp.Status = "success"
	resp.Reply = "flush executed"

	c.JSON(200, resp)

}
