package edition

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	pkgauth "github.com/seantywork/gomehub/pkg/auth"
	"github.com/seantywork/gomehub/pkg/com"
	"github.com/seantywork/gomehub/pkg/dbquery"
	pkgdbq "github.com/seantywork/gomehub/pkg/dbquery"
	pkgutils "github.com/seantywork/gomehub/pkg/utils"
)

func PostVideoUpload(c *gin.Context) {

	_, my_type, my_id := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("video upload: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	file, _ := c.FormFile("file")

	f_name := file.Filename

	f_name_list := strings.Split(f_name, ".")

	f_name_len := len(f_name_list)

	if f_name_len < 1 {

		fmt.Println("no extension specified")

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	v_fname := pkgauth.SanitizePlainNameValue(f_name_list[0])

	extension := f_name_list[f_name_len-1]

	if !pkgutils.CheckIfSliceContains[string](EXTENSION_ALLOWLIST, extension) {

		fmt.Println("extension not allowed")

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	fmt.Printf("received: %s, size: %d\n", file.Filename, file.Size)

	file_name, _ := pkgutils.GetRandomHex(32)

	err := pkgdbq.UploadVideo(c, my_id, file, v_fname, file_name, extension)

	if err != nil {

		fmt.Println(err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to save"})

		return

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: "uploaded"})

}

func PostVideoDelete(c *gin.Context) {

	_, my_type, _ := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("video delete: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	fmt.Println("delete video")

	var req com.CLIENT_REQ

	if err := c.BindJSON(&req); err != nil {

		fmt.Printf("video delete: failed to bind: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	if !pkgauth.VerifyCodeNameValue(req.Data) {

		fmt.Printf("video name verification failed: %s\n", req.Data)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	err := dbquery.DeleteVideo(req.Data)

	if err != nil {

		fmt.Printf("video delete: %s\n", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed delete"})

		return

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: "deleted"})

}

func GetVideoContentByID(c *gin.Context) {

	watchId := c.Param("contentId")

	if !pkgauth.VerifyCodeNameValue(watchId) {

		fmt.Printf("download video: illegal: %s\n", watchId)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	err := pkgdbq.DownloadVideo(c, watchId)

	if err != nil {

		fmt.Printf("download video: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	fmt.Println("video download success")

}

func PostImageUpload(c *gin.Context) {

	_, my_type, my_id := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("image upload: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	file, _ := c.FormFile("file")

	f_name := file.Filename

	f_name_list := strings.Split(f_name, ".")

	f_name_len := len(f_name_list)

	if f_name_len < 1 {

		fmt.Println("no extension specified")

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	v_fname := pkgauth.SanitizePlainNameValue(f_name_list[0])

	extension := f_name_list[f_name_len-1]

	if !pkgutils.CheckIfSliceContains[string](EXTENSION_ALLOWLIST, extension) {

		fmt.Println("extension not allowed")

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	fmt.Printf("received: %s, size: %d\n", file.Filename, file.Size)

	file_name, _ := pkgutils.GetRandomHex(32)

	err := dbquery.UploadImage(c, my_id, file, v_fname, file_name, extension)

	if err != nil {

		fmt.Println(err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to save"})

		return

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: file_name})

}

func GetImageContentById(c *gin.Context) {

	watchId := c.Param("contentId")

	if !pkgauth.VerifyCodeNameValue(watchId) {

		fmt.Printf("download image: illegal: %s\n", watchId)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	err := dbquery.DownloadImage(c, watchId)

	if err != nil {

		fmt.Printf("download image: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	fmt.Println("image download success")
}
