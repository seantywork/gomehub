package edition

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	pkgauth "github.com/seantywork/gomehub/pkg/auth"
	"github.com/seantywork/gomehub/pkg/com"
	"github.com/seantywork/gomehub/pkg/dbquery"
	_ "github.com/seantywork/gomehub/pkg/dbquery"
	pkgutils "github.com/seantywork/gomehub/pkg/utils"
)

var EXTENSION_ALLOWLIST []string

type ArticleInfo struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func PostArticleUpload(c *gin.Context) {

	_, my_type, my_id := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("article upload: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	var req com.CLIENT_REQ

	var a_info ArticleInfo

	if err := c.BindJSON(&req); err != nil {

		fmt.Printf("article upload: failed to bind: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	err := json.Unmarshal([]byte(req.Data), &a_info)

	if err != nil {

		fmt.Printf("article upload: failed to unmarshal: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	new_file_name, _ := pkgutils.GetRandomHex(32)

	plain_name := pkgauth.SanitizePlainNameValue(a_info.Title)

	err = dbquery.UploadArticle(my_id, a_info.Content, plain_name, new_file_name)

	if err != nil {

		fmt.Printf("article upload: failed to upload: %s", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to upload"})

		return
	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: "uploaded"})

}

func PostArticleDelete(c *gin.Context) {

	_, my_type, _ := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("article delete: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	fmt.Println("delete article")

	var req com.CLIENT_REQ

	if err := c.BindJSON(&req); err != nil {

		fmt.Printf("article delete: failed to bind: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	if !pkgauth.VerifyCodeNameValue(req.Data) {

		fmt.Printf("article name verification failed: %s\n", req.Data)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	err := dbquery.DeleteArticle(req.Data)

	if err != nil {

		fmt.Printf("article delete: %s\n", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed delete"})

		return

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: "deleted"})

}

func GetArticleContentById(c *gin.Context) {

	pkgauth.WhoAmI(c)

	watchId := c.Param("contentId")

	if !pkgauth.VerifyCodeNameValue(watchId) {

		fmt.Printf("get article: illegal: %s\n", watchId)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	content, err := dbquery.GetArticle(watchId)

	if err != nil {

		fmt.Printf("failed to get article: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: content})

}

func PostMediaUpload(c *gin.Context) {

	_, my_type, my_id := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("media upload: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	file, _ := c.FormFile("file")

	rawMediaType := file.Header.Get("Content-Type")

	mediaProplist := strings.Split(rawMediaType, "/")

	mediaType := mediaProplist[0]
	mediaExt := mediaProplist[1]

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

	fmt.Printf("received: %s, size: %d, type: %s\n", file.Filename, file.Size, rawMediaType)

	file_name, _ := pkgutils.GetRandomHex(32)

	var err error

	if mediaType == "image" {

		err = dbquery.UploadImage(c, my_id, file, v_fname, file_name, mediaExt)

	} else if mediaType == "video" {

		err = dbquery.UploadVideo(c, my_id, file, v_fname, file_name, mediaExt)

	}

	if err != nil {

		fmt.Println(err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to save"})

		return

	}

	client_file_name := file_name + "." + mediaExt

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: client_file_name})

}

func GetMediaContentById(c *gin.Context) {

	pkgauth.WhoAmI(c)

	watchId := c.Param("contentId")

	check, san := pkgauth.VerifyCodeNameValueWithStop(watchId, '.')

	if !check {

		fmt.Printf("download media: illegal: %s\n", watchId)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	err := dbquery.DownloadMedia(c, san)

	if err != nil {

		fmt.Printf("download media: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	fmt.Println("media download success")
}
