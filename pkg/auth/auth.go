package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/seantywork/gomehub/pkg/com"
	"github.com/seantywork/gomehub/pkg/dbquery"
	pkgutils "github.com/seantywork/gomehub/pkg/utils"
)

var DEBUG bool = false

var USE_OAUTH2 bool = false

var ADMINS = make(map[string]string)

type UserCreate struct {
	Passphrase      string `json:"passphrase"`
	Id              string `json:"id"`
	DurationSeconds int    `json:"duration_seconds"`
}

type UserLogin struct {
	Id         string `json:"id"`
	Passphrase string `json:"passphrase"`
}

type UserListStruct struct {
	Users []string `json:"users"`
}

func GenerateStateAuthCookie(c *gin.Context) string {

	state, _ := pkgutils.GetRandomHex(64)

	session := sessions.Default(c)

	session.Set("SOLIAGAIN", state)
	session.Save()

	return state
}

func RegisterAdmins(admins map[string]string) error {

	err := os.RemoveAll("./data/admin")

	if err != nil {
		return fmt.Errorf("failed to remove data/admin")
	}

	err = os.MkdirAll("./data/admin", 0755)

	if err != nil {

		return fmt.Errorf("failed to create data/admin")
	}

	for k, v := range admins {

		ADMINS[k] = v

		name := "./data/admin/" + k + ".json"

		err := os.WriteFile(name, []byte("{}"), 0644)

		if err != nil {

			return fmt.Errorf("failed to create data/admin: %s: %s", k, err.Error())
		}

	}

	return nil

}

func OauthGoogleLogin(c *gin.Context) {

	my_key, my_type, _ := WhoAmI(c)

	if my_key != "" && my_type != "" {

		fmt.Printf("oauth login: already logged in\n")

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "already logged in"})

		return

	}

	if !USE_OAUTH2 {

		c.Redirect(302, "/signin/idiot")

		return
	}

	oauth_state := GenerateStateAuthCookie(c)

	u := GoogleOauthConfig.AuthCodeURL(oauth_state)

	c.Redirect(302, u)

}

func OauthGoogleCallback(c *gin.Context) {

	my_key, my_type, _ := WhoAmI(c)

	if my_key != "" && my_type != "" {

		fmt.Printf("oauth callback: already logged in\n")

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "already logged in"})

		return

	}

	session := sessions.Default(c)

	var session_id string

	v := session.Get("SOLIAGAIN")

	if v == nil {
		fmt.Printf("access auth failed: %s\n", "session id not found")
		return
	} else {
		session_id = v.(string)
	}

	state := c.Request.FormValue("state")

	if state == "" {
		fmt.Printf("access auth failed: %s\n", "form state not found")
		return
	}

	if state != session_id {
		fmt.Printf("access auth failed: %s\n", "value not matching")
		c.Redirect(302, "/signin")
		return
	}

	data, err := GetUserDataFromGoogle(c.Request.FormValue("code"))
	if err != nil {
		fmt.Printf("access auth failed: %s\n", err.Error())
		c.Redirect(302, "/signin")
		return
	}

	var oauth_struct OAuthStruct

	err = json.Unmarshal(data, &oauth_struct)

	if err != nil {
		fmt.Printf("access auth failed: %s\n", err.Error())
		c.Redirect(302, "/signin")
		return
	}

	if !oauth_struct.VERIFIED_EMAIL {
		fmt.Printf("access auth failed: %s\n", err.Error())
		c.Redirect(302, "/signin")
		return
	}

	if err != nil {
		fmt.Printf("access auth failed: %s\n", err.Error())
		c.Redirect(302, "/signin")
		return
	}

	as, err := dbquery.GetByIdFromAdmin(oauth_struct.EMAIL)

	if as == nil {

		fmt.Printf("access auth failed: %s", err.Error())

		c.Redirect(302, "/signin")

		return

	}

	err = dbquery.MakeSessionForAdmin(session_id, oauth_struct.EMAIL)

	if err != nil {

		fmt.Printf("make session failed for admin: %s", err.Error())

		c.Redirect(302, "/signin")

		return

	}

	fmt.Println("oauth sign in success")

	c.Redirect(302, "/")

}

func UserList(c *gin.Context) {

	_, my_type, _ := WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("user list: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	var uls UserListStruct

	us, err := dbquery.GetAllUsers()

	if err != nil {

		fmt.Printf("user list: failed to get all: %s\n", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to get all users"})

		return

	}

	for k, _ := range us {

		uls.Users = append(uls.Users, k)

	}

	jb, err := json.Marshal(uls)

	if err != nil {

		fmt.Printf("user list: failed to marshal: %s\n", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to get all users"})

		return

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: string(jb)})

}

func UserAdd(c *gin.Context) {

	_, my_type, _ := WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("user add: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	var req com.CLIENT_REQ

	var u_create UserCreate

	if err := c.BindJSON(&req); err != nil {

		fmt.Printf("user add: failed to bind: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	err := json.Unmarshal([]byte(req.Data), &u_create)

	if err != nil {

		fmt.Printf("user add: failed to unmarshal: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	if !VerifyDefaultValue(u_create.Id) {

		fmt.Printf("user add: invalid id: %s\n", u_create.Id)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	err = dbquery.MakeUser(u_create.Id, u_create.Passphrase, u_create.DurationSeconds)

	if err != nil {

		fmt.Printf("user add: failed to make: %s", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to make user"})

		return
	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: fmt.Sprintf("id: %s: created", u_create.Id)})

}

func UserRemove(c *gin.Context) {

	_, my_type, _ := WhoAmI(c)

	if my_type != "admin" {

		fmt.Printf("user remove: not admin\n")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "you're not admin"})

		return

	}

	var req com.CLIENT_REQ

	if err := c.BindJSON(&req); err != nil {

		fmt.Printf("user remove: failed to bind: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	if !VerifyDefaultValue(req.Data) {

		fmt.Printf("user remove: invalid id: %s\n", req.Data)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	err := dbquery.RemoveUser(req.Data)

	if err != nil {

		fmt.Printf("user remove: failed to remove: %s", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to remove user"})

		return
	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: fmt.Sprintf("id: %s: deleted", req.Data)})

}

func Login(c *gin.Context) {

	my_key, my_type, my_id := WhoAmI(c)

	if my_key != "" && my_type != "" {

		dbquery.RemoveSessionKeyFromSession(my_key)

		fmt.Printf("user login: removed existing id: %s\n", my_id)

		return

	}

	var req com.CLIENT_REQ

	var u_login UserLogin

	if err := c.BindJSON(&req); err != nil {

		fmt.Printf("user login: failed to bind: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	err := json.Unmarshal([]byte(req.Data), &u_login)

	if err != nil {

		fmt.Printf("user login: failed to unmarshal: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	if !VerifyDefaultValue(u_login.Id) {

		fmt.Printf("user login: not valid id: %s\n", u_login.Id)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	us, err := dbquery.GetByIdFromUser(u_login.Id)

	if err != nil {

		fmt.Printf("user login: failed to get from user: %s\n", err.Error())

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "id doesn't exist"})

		return
	}

	if us.Passphrase != u_login.Passphrase {

		fmt.Printf("user login: passphrase: %s", "not matching")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "passphrase not matching"})

		return

	}

	session_key := GenerateStateAuthCookie(c)

	err = dbquery.MakeSessionForUser(session_key, u_login.Id, us.DurationSeconds)

	if err != nil {

		fmt.Printf("user login: failed to get from user: %s", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to login"})

		return

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: "logged in"})

}

func Logout(c *gin.Context) {

	my_key, my_type, _ := WhoAmI(c)

	if my_key == "" && my_type == "" {

		fmt.Printf("user logout: not logged in\n")

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "not logged in"})

		return

	}

	if my_type == "admin" || my_type == "user" {

		err := dbquery.RemoveSessionKeyFromSession(my_key)

		if err != nil {

			fmt.Printf("user logout: failed to remove session key: %s", err.Error())

			c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to logout"})

			return

		}

	} else {

		fmt.Printf("user logout: what the hell is this type?: %s", my_type)

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to logout"})

		return

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: "logged out"})

}

func LoginIdiot(c *gin.Context) {

	my_key, my_type, my_id := WhoAmI(c)

	if my_key != "" && my_type != "" {

		dbquery.RemoveSessionKeyFromSession(my_key)

		fmt.Printf("user login: removed existing id: %s\n", my_id)

		return

	}

	var req com.CLIENT_REQ

	var u_login UserLogin

	if err := c.BindJSON(&req); err != nil {

		fmt.Printf("user login: failed to bind: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	err := json.Unmarshal([]byte(req.Data), &u_login)

	if err != nil {

		fmt.Printf("user login: failed to unmarshal: %s\n", err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return

	}

	if !VerifyDefaultValue(u_login.Id) {

		fmt.Printf("user login: not valid id: %s\n", u_login.Id)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	as, err := dbquery.GetByIdFromAdmin(u_login.Id)

	if as == nil {

		fmt.Printf("user login: no such admin id: %s: %s\n", u_login.Id, err.Error())

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	credPw, okay := ADMINS[u_login.Id]

	if !okay {

		fmt.Printf("user login: no such admin id in admins: %s\n", u_login.Id)

		c.JSON(http.StatusBadRequest, com.SERVER_RE{Status: "error", Reply: "invalid format"})

		return
	}

	if u_login.Passphrase != credPw {

		fmt.Printf("user login: passphrase: %s", "not matching")

		c.JSON(http.StatusForbidden, com.SERVER_RE{Status: "error", Reply: "passphrase not matching"})

		return

	}

	session_key := GenerateStateAuthCookie(c)

	err = dbquery.MakeSessionForAdmin(session_key, u_login.Id)

	if err != nil {

		fmt.Printf("user login: failed to get from user: %s", err.Error())

		c.JSON(http.StatusInternalServerError, com.SERVER_RE{Status: "error", Reply: "failed to login"})

		return

	}

	c.JSON(http.StatusOK, com.SERVER_RE{Status: "success", Reply: "logged in"})

}
