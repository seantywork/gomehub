package controller

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	pkgauth "github.com/seantywork/gomehub/pkg/auth"
	pkgcom "github.com/seantywork/gomehub/pkg/com"
	pkgedition "github.com/seantywork/gomehub/pkg/edition"
	pkglog "github.com/seantywork/gomehub/pkg/log"
	pkgman "github.com/seantywork/gomehub/pkg/manage"
	pkgstream "github.com/seantywork/gomehub/pkg/stream"
	pkgutils "github.com/seantywork/gomehub/pkg/utils"
)

func CreateServer() *gin.Engine {

	genserver := gin.Default()

	store := sessions.NewCookieStore([]byte("SOLIAGAIN"))

	genserver.Use(sessions.Sessions("SOLIAGAIN", store))

	ConfigureRuntime(genserver)

	RegisterRoutes(genserver)

	return genserver

}

func ConfigureRuntime(e *gin.Engine) {

	e.MaxMultipartMemory = CONF.MaxFileSize

	//pkgsoli.DEBUG = CONF.Debug

	//pkgsoli.SOLI_FRONT_ADDR = CONF.Sorrylinus.FrontAddr

	//pkgsoli.SOLI_SIGNAL_PATH = CONF.Sorrylinus.SoliSignalAddr

	//pkgsoli.TIMEOUT_SEC = CONF.TimeoutSec

	//pkgsoli.EXTERNAL_URL = CONF.ExternalUrl

	//pkgsoli.INTERNAL_URL = CONF.InternalUrl

	pkgauth.DEBUG = CONF.Debug
	pkgauth.USE_OAUTH2 = CONF.Auth.UseOauth2

	adminslen := len(CONF.Auth.Admins)

	admins := make(map[string]string)

	for i := 0; i < adminslen; i++ {

		admins[CONF.Auth.Admins[i].Id] = CONF.Auth.Admins[i].Pw

	}

	err := pkgauth.RegisterAdmins(admins)

	if err != nil {

		log.Fatalf("failed to register admins: %s", err.Error())

		return
	}

	pkgcom.CHANNEL_ADDR = CONF.ServeAddr
	pkgcom.CHANNEL_PORT = fmt.Sprintf("%d", CONF.Com.ChannelPort)
	pkgcom.CHANNEL_PORT_EXTERNAL = fmt.Sprintf("%d", CONF.Com.ChannelPortExternal)

	pkgedition.EXTENSION_ALLOWLIST = CONF.Edition.ExtAllowList

	pkgstream.DEBUG = CONF.Debug

	for i := 0; i < len(CONF.Stream.TurnServerAddr); i++ {

		tmp := struct {
			Addr string `json:"addr"`
			Id   string `json:"id"`
			Pw   string `json:"pw"`
		}{
			Addr: CONF.Stream.TurnServerAddr[i].Addr,
			Id:   CONF.Stream.TurnServerAddr[i].Id,
			Pw:   CONF.Stream.TurnServerAddr[i].Pw,
		}

		pkgstream.TURN_SERVER_ADDR = append(pkgstream.TURN_SERVER_ADDR, tmp)
	}

	pkgstream.EXTERNAL_URL = CONF.ExternalUrl

	pkgstream.INTERNAL_URL = CONF.InternalUrl

	pkgstream.TIMEOUT_SEC = CONF.TimeoutSec

	pkgstream.PEER_SIGNAL_ATTEMPT_SYNC = CONF.Stream.PeerSignalAttemptSync

	pkgstream.PEERS_SIGNAL_PATH = CONF.Stream.PeerSignalAddr

	pkgstream.RTCP_PLI_INTERVAL = time.Second * time.Duration(CONF.Stream.RtcpPLIInterval)

	pkgstream.SINGLE_ROOM_MODE = CONF.Stream.SingleRoomMode

	pkgstream.UDP_BUFFER_BYTE_SIZE = CONF.Stream.UdpBufferByteSize
	pkgstream.UDP_MUX_PORT = CONF.Stream.UdpMuxPort
	pkgstream.UDP_EPHEMERAL_PORT_MIN = CONF.Stream.UdpEphemeralPortMin
	pkgstream.UDP_EPHEMERAL_PORT_MAX = CONF.Stream.UdpEphemeralPortMax

	pkgstream.RTP_RECEIVE_ADDR = CONF.ServeAddr
	pkgstream.RTP_RECEIVE_PORT = fmt.Sprintf("%d", CONF.Stream.RtpReceivePort)
	pkgstream.RTP_RECEIVE_PORT_EXTERNAL = fmt.Sprintf("%d", CONF.Stream.RtpReceivePortExternal)

	pkglog.FLUSH_INTERVAL_SEC = CONF.Log.FlushIntervalSec

	pkgutils.USE_COMPRESS = CONF.Utils.UseCompress

}

func RegisterRoutes(e *gin.Engine) {

	// base

	e.LoadHTMLGlob("view/**/*")

	e.Static("/public", "./public")

	e.GET("/", GetIndex)

	e.GET("/signin", GetViewSignin)

	e.GET("/signin/idiot", GetViewSigninIdiot)

	e.GET("/mypage", GetViewMypage)

	e.GET("/mypage/article", GetViewMypageArticle)

	// e.GET("/mypage/video", GetViewMypageVideo)

	e.GET("/mypage/room", GetViewMypageRoom)

	e.GET("/content/article/:articleId", GetViewContentArticle)

	// e.GET("/content/video/:videoId", GetViewContentVideo)

	e.GET("/room/:roomId", GetViewRoom)

	e.GET("/api/content/entry", GetMediaEntry)

	// auth

	e.GET("/api/oauth2/google/signin", pkgauth.OauthGoogleLogin)

	e.GET("/oauth2/google/callback", pkgauth.OauthGoogleCallback)

	e.POST("/api/auth/signin/idiot", pkgauth.LoginIdiot)

	e.GET("/api/auth/user/list", pkgauth.UserList)

	e.POST("/api/auth/user/add", pkgauth.UserAdd)

	e.POST("/api/auth/user/remove", pkgauth.UserRemove)

	e.POST("/api/auth/signin", pkgauth.Login)

	e.GET("/api/auth/signout", pkgauth.Logout)

	pkgauth.InitAuth()

	// sorrylinus

	//e.POST("/api/sorrylinus/open", pkgsoli.PostSoliOpen)

	//e.POST("/api/sorrylinus/close", pkgsoli.PostSoliClose)

	// e.GET("/api/sorrylinus/signal/address", pkgsoli.GetSoliSignalAddress)

	// edition

	e.POST("/api/article/upload", pkgedition.PostArticleUpload)

	e.POST("/api/article/delete", pkgedition.PostArticleDelete)

	e.GET("/api/article/c/:contentId", pkgedition.GetArticleContentById)

	e.POST("/api/media/upload", pkgedition.PostMediaUpload)

	e.GET("/api/media/c/:contentId", pkgedition.GetMediaContentById)

	// e.POST("/api/video/upload", pkgedition.PostVideoUpload)

	// e.POST("/api/video/delete", pkgedition.PostVideoDelete)

	//e.GET("/api/video/c/:contentId", pkgedition.GetVideoContentByID)

	// stream

	pkgstream.InitWebRTCApi()

	e.POST("/api/cctv/open", pkgstream.PostCCTVOpen)

	e.POST("/api/cctv/close", pkgstream.PostCCTVClose)

	go pkgstream.InitRTMPServer()

	e.GET("/api/peers/entry", pkgstream.GetPeersEntry)

	e.POST("/api/peers/create", pkgstream.PostPeersCreate)

	e.POST("/api/peers/delete", pkgstream.PostPeersDelete)

	e.GET("/api/peers/signal/address", pkgstream.GetPeersSignalAddress)

	// com

	//pkgcom.AddChannelHandler(CONF.Sorrylinus.SoliSignalAddr, pkgsoli.SoliSignalHandler)

	if pkgstream.SINGLE_ROOM_MODE {
		pkgcom.AddChannelHandler(CONF.Stream.PeerSignalAddr, pkgstream.RoomSignalHandlerSingle)

		pkgcom.AddChannelCallback(pkgstream.SignalDispatcherSingle)

	} else {
		pkgcom.AddChannelHandler(CONF.Stream.PeerSignalAddr, pkgstream.RoomSignalHandler)

		pkgcom.AddChannelCallback(pkgstream.SignalDispatcher)
	}

	go pkgcom.StartAllChannelHandlers()

	// manage

	e.GET("/api/manage/log/flush", pkgman.GetManualLogFlush)

	// log

	pkglog.InitLog()
}
