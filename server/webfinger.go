package server

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/ngerakines/tavern/model"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

type UserLookup func(*gorm.DB, string, string) (bool, error)

type WebfingerHandler struct {
	Domain string
	Logger *zap.Logger
	DB     *gorm.DB
}

type webFingerResponse struct {
	Subject string                  `json:"subject"`
	Aliases []string                `json:"aliases"`
	Links   []webFingerLinkResponse `json:"links"`
}

type webFingerLinkResponse struct {
	Rel      string `json:"rel"`
	LinkType string `json:"type"`
	HREF     string `json:"href"`
}

func createWebFingerResponse(user, domain string) webFingerResponse {
	return webFingerResponse{
		Subject: fmt.Sprintf("%s@%s", user, domain),
		Aliases: []string{fmt.Sprintf("%s@%s", user, domain)},
		Links: []webFingerLinkResponse{
			{
				Rel:      "self",
				LinkType: "application/activity+json",
				HREF:     fmt.Sprintf("https://%s/users/%s", domain, user),
			},
		},
	}
}

func (h WebfingerHandler) Webfinger(c *gin.Context) {
	user, domain, err := fingerUserDomain(c.Query("resource"), h.Domain)
	if err != nil {
		h.Logger.Error("failed parsing resource", zap.Error(err), zap.String("resource", c.Query("resource")))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	ok, err := model.ActorLookup(h.DB, user, domain)
	if err != nil {
		h.Logger.Error("failed looking up user", zap.Error(err), zap.String("user", user), zap.String("domain", domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !ok {
		h.Logger.Error("user not found", zap.String("user", user), zap.String("domain", domain))
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	c.Writer.Header().Set("Content-Type", "application/jrd+json")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Pragma", "no-cache")
	c.JSON(200, createWebFingerResponse(user, domain))
}

func fingerUserDomain(input, domain string) (string, string, error) {
	input = strings.TrimPrefix(input, "acct:")
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == '@'
	})
	if len(parts) != 2 {
		return "", "", errors.New("malformed resource parameter")
	}
	if parts[1] != domain {
		return "", "", errors.New("malformed resource parameter")
	}
	user := strings.TrimSpace(parts[0])
	if len(user) == 0 {
		return "", "", errors.New("malformed resource parameter")
	}
	return user, domain, nil
}
