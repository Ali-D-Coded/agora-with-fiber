package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/AgoraIO-Community/go-tokenbuilder/rtctokenbuilder"
	"github.com/AgoraIO-Community/go-tokenbuilder/rtmtokenbuilder"
	"github.com/gofiber/fiber/v2"
)
var APPID, APP_CERTIFICATE string 
func main() {
	fmt.Println("Agora Token Builder")
// 	os.Setenv("APP_ID", "18aa7610b5a94be68a09484435b3e780")
// 	os.Setenv("APP_CERTIFICATE", "23f2f14910b2499a980ecaf579ff61de")

	appIDEnv, appIDExists := os.LookupEnv("APP_ID")
	appCertEnv, appCertExists := os.LookupEnv("APP_CERTIFICATE")

	if !appIDExists || !appCertExists {
		log.Fatal("FATAL ERROR : ENV not properly configured, check APP_ID and APP_CERTIFICATE")
	} else {
		APPID = appIDEnv
		APP_CERTIFICATE = appCertEnv
	}

	api := fiber.New()
	api.Static("/", "./public")

	// api.Get("/", func(ctx *fiber.Ctx) error {
	// 	return ctx.SendString("Helo World")
	// })
	api.Get("/ping", func(ctx *fiber.Ctx) error {
		return ctx.JSON(fiber.Map{
			"message": "pong",
		})
	})
	api.Get("/envs", func(ctx *fiber.Ctx) error {
		return ctx.JSON(fiber.Map{
			"appid":APPID, "apcert":APP_CERTIFICATE,
		})
	})

	api.Get("/rtc/:channelName/:role/:tokenType/:uid", getRtcToken)
	api.Get("/rtm/:uid", getRtmToken)
	api.Get("/rte/:channelName/:role/:tokenType/:uid", getBothTokens)

	port := os.Getenv("PORT")

	if port == "" {
		port = "3000"
	}

    log.Fatal(api.Listen("0.0.0.0:" + port))
}


func getRtcToken(c *fiber.Ctx) error {
	// get param values
	channelName, tokenType, uidStr, role, expireTimeStamp, err := parseRTCParams(c)
	fmt.Println("err :",err)

	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Error Generating RTC Token: " + err.Error(),
			"status":  400,
		})
	}

	// generate the token
	rtcToken, tokenErr := generateRTCToken(channelName, uidStr, tokenType, role, expireTimeStamp)

	// return the token in JSON response
	if tokenErr != nil {
		log.Println(tokenErr)
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"status":  400,
			"message": "Error Generating RTC token: " + tokenErr.Error(),
		})
	} else {
		return c.JSON(fiber.Map{
			"rtcToken": rtcToken,
		})
	}
}

func getRtmToken(c *fiber.Ctx) error {
	// get param values
	uidStr, expireTimeStamp, err := parseRTMParams(c)
	fmt.Println("expire :",expireTimeStamp)
	fmt.Println("err", err)

	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"status":  400,
			"message": "Error Generating rtm token",
			"err":err,
		})
	}

	// build rtm token
	rtmToken, tokenErr := rtmtokenbuilder.BuildToken(APPID, APP_CERTIFICATE, uidStr, rtmtokenbuilder.RoleRtmUser, expireTimeStamp)

	// return rtm token
	if tokenErr != nil {
		log.Println(tokenErr)
		errMsg := "Error Generating RTM Token : " + tokenErr.Error()
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"status": 400,
			"error":  errMsg,
		})
	} else {
		return c.JSON(fiber.Map{
			"rtmToken": rtmToken,
})
}
}

func getBothTokens(c *fiber.Ctx) error {
// get the params
channelName, tokenType, uidStr, role, expireTimeStamp, rtcParamErr := parseRTCParams(c)
if rtcParamErr != nil {
c.Status(fiber.StatusBadRequest)
return c.JSON(fiber.Map{
"status": 400,
"message": "Error generating tokens: " + rtcParamErr.Error(),
})
}
// generate rtc token
rtcToken, rtcTokenErr := generateRTCToken(channelName, uidStr, tokenType, role, expireTimeStamp)
// generate rtm token
rtmToken, rtmTokenErr := rtmtokenbuilder.BuildToken(APPID, APP_CERTIFICATE, uidStr, rtmtokenbuilder.RoleRtmUser, expireTimeStamp)// return both tokens
if rtcTokenErr != nil {
	c.Status(fiber.StatusBadRequest)
	errMsg := "Error generating RTC Token: " + rtcTokenErr.Error()
	return c.JSON(fiber.Map{
		"status":  400,
		"message": errMsg,
	})
} else if rtmTokenErr != nil {
	c.Status(fiber.StatusBadRequest)
	errMsg := "Error generating RTM Token: " + rtmTokenErr.Error()
	return c.JSON(fiber.Map{
		"status":  400,
		"message": errMsg,
	})
} else {
	return c.JSON(fiber.Map{
		"rtcToken": rtcToken,
		"rtmToken": rtmToken,
	})
}
}

type CustomError string

func (e CustomError) Error() string {
	return string(e)
}

const (
	RoleUndefined   rtctokenbuilder.Role = iota
	RolePublisher
	RoleSubscriber
)



func parseRTCParams(c *fiber.Ctx) (channelName, tokenType, uidStr string, role rtctokenbuilder.Role, expireTimeStamp uint32, err error) {
	channelName = c.Params("channelName")
	roleStr := c.Params("role")
	tokenType = c.Params("tokenType")
	uidStr = c.Params("uid")
	expireTime := c.Query("expiry")

	if expireTime == "" {
		expireTime = "3600"
	}

	expireTime64, parseErr := strconv.ParseUint(expireTime, 10, 64)
	if parseErr != nil {
		err = fmt.Errorf("failed to parse expireTime: %s, causing error: %s", expireTime, parseErr)
		return "", "", "", RoleUndefined, 0, err
	}

	expireTimeInSeconds := uint32(expireTime64)
	currentTimeStamp := uint32(time.Now().UTC().Unix())
	expireTimeStamp = currentTimeStamp + expireTimeInSeconds

	if roleStr == "publisher" {
		role = RolePublisher
	} else {
		role = RoleSubscriber
	}

	return channelName, tokenType, uidStr, role, expireTimeStamp, nil
}



func parseRTMParams(c *fiber.Ctx) (uidStr string, expireTimeStamp uint32, err error) {
// get param values
uidStr = c.Params("uid")
expireTime := c.Query("expiry")

	if expireTime == "" {
		expireTime = "3600"
	}
expireTime64, parseErr := strconv.ParseUint(expireTime, 10, 64)

if parseErr != nil {
	err = fmt.Errorf("failed to parse expireTime: %s, causing error: %s", expireTime, parseErr)
}

expireTimeInSeconds := uint32(expireTime64)
currentTimeStamp := uint32(time.Now().UTC().Unix())
expireTimeStamp = currentTimeStamp + expireTimeInSeconds

return uidStr, expireTimeStamp, err 
}

func generateRTCToken(channelName, uidStr, tokenType string, role rtctokenbuilder.Role, expireTimeStamp uint32) (rtcToken string, err error) {
	// Check token type
	if tokenType == "userAccount" {
		rtcToken, err = rtctokenbuilder.BuildTokenWithUserAccount(APPID, APP_CERTIFICATE, channelName, uidStr, role, expireTimeStamp)
	} else if tokenType == "uid" {
		uid64, parseErr := strconv.ParseUint(uidStr, 10, 64)
		if parseErr != nil {
			err = fmt.Errorf("failed to parse uidStr: %s, causing error: %s", uidStr, parseErr)
			return "", err
		}
		uid := uint32(uid64)
		rtcToken, err = rtctokenbuilder.BuildTokenWithUID(APPID, APP_CERTIFICATE, channelName, uid, role, expireTimeStamp)
	} else {
		err = fmt.Errorf("failed to generate RTC token for unknown tokenType: %s", tokenType)
		log.Println(err)
		return "", err
	}

	if err != nil {
		err = fmt.Errorf("failed to generate RTC token: %s", err)
	}

	return rtcToken, err
}
