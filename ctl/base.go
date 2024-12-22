package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pkgauth "github.com/seantywork/gomehub/pkg/auth"
	"github.com/seantywork/gomehub/pkg/com"
	"github.com/seantywork/gomehub/pkg/dbquery"
	pkgstream "github.com/seantywork/gomehub/pkg/stream"
)

type EntryStruct struct {
	Entry []struct {
		Title     string    `json:"title"`
		Id        string    `json:"id"`
		Type      string    `json:"type"`
		Timestamp time.Time `json:"timestamp"`
		Author    string    `json:"author"`
	} `json:"entry"`
}

func GetIndex(c *gin.Context) {

	my_key, my_type, _ := pkgauth.WhoAmI(c)

	if my_key == "" && my_type == "" {

		c.HTML(200, "index/index.html", gin.H{})

	} else {

		c.HTML(200, "index/index.html", gin.H{
			"logged_in": true,
		})
	}

}

func GetViewSignin(c *gin.Context) {

	pkgauth.WhoAmI(c)

	c.HTML(200, "index/signin.html", gin.H{})

}

func GetViewSigninIdiot(c *gin.Context) {

	pkgauth.WhoAmI(c)

	c.HTML(200, "index/signinidiot.html", gin.H{})

}

func GetViewMypage(c *gin.Context) {

	_, my_type, _ := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("view my page: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	c.HTML(200, "mypage/index.html", gin.H{})

}

func GetViewMypageArticle(c *gin.Context) {

	_, my_type, _ := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("view my page: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	c.HTML(200, "mypage/article.html", gin.H{})

}

func GetViewMypageVideo(c *gin.Context) {

	_, my_type, _ := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("view my page: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	c.HTML(200, "mypage/video.html", gin.H{})

}

func GetViewMypageRoom(c *gin.Context) {

	_, my_type, _ := pkgauth.WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("view my page: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	c.HTML(200, "mypage/room.html", gin.H{})

}

func GetViewContentArticle(c *gin.Context) {

	pkgauth.WhoAmI(c)

	watchId := c.Param("articleId")

	if !pkgauth.VerifyCodeNameValue(watchId) {

		fmt.Printf("get article: illegal: %s\n", watchId)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	c.HTML(200, "content/article.html", gin.H{
		"article_code": watchId,
	})

}

func GetViewContentVideo(c *gin.Context) {

	watchId := c.Param("videoId")

	if !pkgauth.VerifyCodeNameValue(watchId) {

		fmt.Printf("get video: illegal: %s\n", watchId)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	c.HTML(200, "content/video.html", gin.H{
		"video_code": watchId,
	})

}

func GetViewRoom(c *gin.Context) {

	my_key, my_type, my_id := pkgauth.WhoAmI(c)

	if my_key == "" && my_type == "" {

		fmt.Printf("view room: not logged in\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "not logged in"})

		return

	}

	watchId := c.Param("roomId")

	p_users, okay := pkgstream.ROOMREG[watchId]

	if !okay {

		fmt.Printf("view room: no such room\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "not allowed"})

		return

	}

	pu_len := len(p_users)

	allowed := 0

	user_index := -1

	for i := 0; i < pu_len; i++ {

		if p_users[i].User == my_id {

			allowed = 1

			user_index = i

			break
		}

	}

	if allowed != 1 {

		fmt.Printf("view room: user not allowed\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "not allowed"})

		return

	}

	var pj pkgstream.PeersJoin

	pj.RoomName = watchId
	pj.User = p_users[user_index].User
	pj.UserKey = p_users[user_index].UserKey

	jb, err := json.Marshal(pj)

	if err != nil {

		fmt.Printf("view room: marshal\n")

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to get room"})

		return

	}

	c.HTML(200, "room/index.html", gin.H{
		"room_code": string(jb),
	})

}

func GetMediaEntry(c *gin.Context) {

	pkgauth.WhoAmI(c)

	entry := EntryStruct{
		Entry: make([]struct {
			Title     string    "json:\"title\""
			Id        string    "json:\"id\""
			Type      string    "json:\"type\""
			Timestamp time.Time "json:\"timestamp\""
			Author    string    "json:\"author\""
		}, 0),
	}

	em, err := dbquery.GetEntryForMedia()

	if err != nil {

		fmt.Printf("get content entry: failed to retrieve: %s\n", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to retrieve content entry"})

		return

	}

	for k, v := range em {

		t, _ := time.Parse("2006-01-02-15-04-05", v.CreatedAt)

		if v.Type == "article" {

			entry.Entry = append(entry.Entry, struct {
				Title     string    `json:"title"`
				Id        string    `json:"id"`
				Type      string    `json:"type"`
				Timestamp time.Time `json:"timestamp"`
				Author    string    `json:"author"`
			}{

				Title:     v.PlainName,
				Id:        k,
				Type:      "article",
				Timestamp: t,
				Author:    v.Author,
			})

		} else {

			continue
		}

	}

	jb, err := json.Marshal(entry)

	if err != nil {

		fmt.Printf("get content entry: failed to marshal: %s\n", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to retrieve content entry"})

		return

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: string(jb)})

}
