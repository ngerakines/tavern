package server

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/ngerakines/tavern/model"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type ActorHandler struct {
	Domain string
	Logger *zap.Logger
	DB     *gorm.DB
}

var (
	followersPerPage = 20
	followingPerPage = 20
	activityPerPage  = 20
)

func (h ActorHandler) ActorHandler(c *gin.Context) {
	user := c.Param("user")

	actor := model.NewActorID(user, h.Domain)

	publicKey, err := model.ActorPublicKey(h.DB, user, h.Domain)
	if err != nil {
		h.Logger.Error("unable to get public key", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	actorContext := []interface{}{
		"https://www.w3.org/ns/activitystreams",
		"https://w3id.org/security/v1",
		map[string]interface{}{
			"featured": map[string]interface{}{
				"@id":   "toot:featured",
				"@type": "@id",
			},
			"alsoKnownAs": map[string]interface{}{
				"@id":   "as:alsoKnownAs",
				"@type": "@id",
			},
			"movedTo": map[string]interface{}{
				"@id":   "as:movedTo",
				"@type": "@id",
			},
			"schema":        "http://schema.org#",
			"PropertyValue": "schema:PropertyValue",
			"value":         "schema:value",
			"IdentityProof": "toot:IdentityProof",
			"discoverable":  "toot:discoverable",
			"focalPoint": map[string]interface{}{
				"@container": "@list",
				"@id":        "toot:focalPoint",
			},
		},
	}
	document := map[string]interface{}{
		"@context":          actorContext,
		"type":              "person",
		"id":                string(actor),
		"name":              user,
		"preferredUsername": user,
		"inbox":             actor.Inbox(),
		"outbox":            actor.Outbox(),
		"followers":         actor.Followers(),
		"following":         actor.Following(),
		"publicKey": map[string]interface{}{
			"id":           "https://mastodon.social/users/ngerakines#main-key",
			"owner":        actor.MainKey(),
			"publicKeyPem": publicKey,
		},
	}

	documentContext := map[string]interface{}{
		"@context": actorContext,
	}

	result, err := compactJSONLD(document, documentContext)
	if err != nil {
		h.Logger.Error("unable to create response", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	WriteJSONLD(c, result)
}

func (h ActorHandler) FollowersHandler(c *gin.Context) {
	if c.Query("page") == "" {
		h.FollowersIndexHandler(c)
		return
	}
	h.FollowersPageHandler(c)
}

func (h ActorHandler) FollowersIndexHandler(c *gin.Context) {
	user := c.Param("user")
	actor := model.NewActorID(user, h.Domain)

	count, err := model.FollowersCount(h.DB, string(model.NewActorID(user, h.Domain)))
	if err != nil {
		h.Logger.Error("unable to get followers", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	rootContext := []interface{}{
		"https://www.w3.org/ns/activitystreams",
	}
	document := map[string]interface{}{
		"@context":   rootContext,
		"type":       "OrderedCollection",
		"id":         actor.Followers(),
		"totalItems": count,
	}
	if count > 0 {
		document["first"] = actor.FollowersPage(1)
	}

	documentContext := map[string]interface{}{
		"@context": rootContext,
	}

	result, err := compactJSONLD(document, documentContext)
	if err != nil {
		h.Logger.Error("unable to create response", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	WriteJSONLD(c, result)
}

func (h ActorHandler) FollowersPageHandler(c *gin.Context) {
	user := c.Param("user")
	actor := model.NewActorID(user, h.Domain)
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil {
		h.Logger.Warn("invalid followers page", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain), zap.String("page", c.Query("page")))
		page = 1
	}
	if page < 1 {
		page = 1
	}

	followers, err := model.FollowersPageLookup(h.DB, string(model.NewActorID(user, h.Domain)), page, followersPerPage)
	if err != nil {
		h.Logger.Error("unable to get followers", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	rootContext := []interface{}{
		"https://www.w3.org/ns/activitystreams",
		map[string]interface{}{
			"schema":        "http://schema.org#",
			"PropertyValue": "schema:PropertyValue",
			"value":         "schema:value",
			"orderedItems": map[string]interface{}{
				"@container": "@list",
				"@id":        "as:orderedItems",
			},
		},
	}
	document := map[string]interface{}{
		"@context":   rootContext,
		"id":         actor.FollowersPage(page),
		"type":       "OrderedCollectionPage",
		"totalItems": len(followers),
		"partOf":     actor.Followers(),
	}
	if len(followers) > 0 {
		document["orderedItems"] = followers
	}
	if len(followers) == followersPerPage {
		document["next"] = actor.FollowersPage(page + 1)
	}
	if page > 1 {
		document["prev"] = actor.FollowersPage(page - 1)
	}

	documentContext := map[string]interface{}{
		"@context": rootContext,
	}

	result, err := compactJSONLD(document, documentContext)
	if err != nil {
		h.Logger.Error("unable to create response", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	WriteJSONLD(c, result)
}

func (h ActorHandler) FollowingHandler(c *gin.Context) {
	if c.Query("page") == "" {
		h.FollowingIndexHandler(c)
		return
	}
	h.FollowingPageHandler(c)
}

func (h ActorHandler) FollowingIndexHandler(c *gin.Context) {
	user := c.Param("user")
	actor := model.NewActorID(user, h.Domain)

	count, err := model.FollowingCount(h.DB, string(model.NewActorID(user, h.Domain)))
	if err != nil {
		h.Logger.Error("unable to get following", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	rootContext := []interface{}{
		"https://www.w3.org/ns/activitystreams",
	}
	document := map[string]interface{}{
		"@context":   rootContext,
		"type":       "OrderedCollection",
		"id":         actor.Following(),
		"totalItems": count,
	}
	if count > 0 {
		document["first"] = actor.FollowingPage(1)
	}

	documentContext := map[string]interface{}{
		"@context": rootContext,
	}

	result, err := compactJSONLD(document, documentContext)
	if err != nil {
		h.Logger.Error("unable to create response", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	WriteJSONLD(c, result)
}

func (h ActorHandler) FollowingPageHandler(c *gin.Context) {
	user := c.Param("user")
	actor := model.NewActorID(user, h.Domain)
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil {
		h.Logger.Warn("invalid following page", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain), zap.String("page", c.Query("page")))
		page = 1
	}
	if page < 1 {
		page = 1
	}

	followers, err := model.FollowingPageLookup(h.DB, string(model.NewActorID(user, h.Domain)), page, followingPerPage)
	if err != nil {
		h.Logger.Error("unable to get followers", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	rootContext := []interface{}{
		"https://www.w3.org/ns/activitystreams",
		map[string]interface{}{
			"schema":        "http://schema.org#",
			"PropertyValue": "schema:PropertyValue",
			"value":         "schema:value",
			"orderedItems": map[string]interface{}{
				"@container": "@list",
				"@id":        "as:orderedItems",
			},
		},
	}
	document := map[string]interface{}{
		"@context":   rootContext,
		"id":         actor.FollowingPage(page),
		"type":       "OrderedCollectionPage",
		"totalItems": len(followers),
		"partOf":     actor.Following(),
	}
	if len(followers) > 0 {
		document["orderedItems"] = followers
	}
	if len(followers) == followingPerPage {
		document["next"] = actor.FollowingPage(page + 1)
	}
	if page > 1 {
		document["prev"] = actor.FollowingPage(page - 1)
	}

	documentContext := map[string]interface{}{
		"@context": rootContext,
	}

	result, err := compactJSONLD(document, documentContext)
	if err != nil {
		h.Logger.Error("unable to create response", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	WriteJSONLD(c, result)
}
