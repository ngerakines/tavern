package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/ngerakines/tavern/model"
	"github.com/piprate/json-gold/ld"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type ActorHandler struct {
	Domain string
	Logger *zap.Logger
	DB     *gorm.DB
}

type collectionResponse struct {
	Context      string   `json:"@context"`
	ResponseType string   `json:"type"`
	ID           string   `json:"id"`
	Total        int64    `json:"totalItems"`
	Items        []string `json:"items,omitempty"`
}

func createFollowersResponse(user, domain string, followers []string) (map[string]interface{}, error) {
	/*
		{
		  "@context": "https://www.w3.org/ns/activitystreams",
		  "type": "Collection",
		  "id": "https://tavern.ngrok.io/users/nick/followers",
		  "totalItems": 0
		}
	*/
	proc := ld.NewJsonLdProcessor()
	options := ld.NewJsonLdOptions("https://www.w3.org/ns/activitystreams")
	options.ProcessingMode = ld.JsonLd_1_1

	actor := model.NewActorID(user, domain)

	doc := map[string]interface{}{
		"@context":   jsonldContextFollowers(),
		"type":       "Collection",
		"id":         actor.Followers(),
		"totalItems": len(followers),
		"items":      followers,
	}

	context := map[string]interface{}{
		"@context": jsonldContextFollowers(),
	}

	return proc.Compact(doc, context, options)
}

func createFollowingResponse(user, domain string, following []string) collectionResponse {
	return collectionResponse{
		Context:      "https://www.w3.org/ns/activitystreams",
		ResponseType: "Collection",
		ID:           fmt.Sprintf("https://%s/users/%s/following", domain, user),
		Total:        int64(len(following)),
		Items:        following,
	}
}

func (h ActorHandler) ActorHandler(c *gin.Context) {
	user := c.Param("user")

	actor := model.NewActorID(user, h.Domain)

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

	followers, err := model.FollowersPageLookup(h.DB, string(model.NewActorID(user, h.Domain)), page, 20)
	if err != nil {
		h.Logger.Error("unable to get followers", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	/*
		{
		  "@context": "https://www.w3.org/ns/activitystreams",
		  "id": "https://mastodon.social/users/ngerakines/following?page=1",
		  "type": "OrderedCollectionPage",
		  "totalItems": 20,
		  "next": "https://mastodon.social/users/ngerakines/following?page=2",
		  "partOf": "https://mastodon.social/users/ngerakines/following",
		  "orderedItems": [
		    "https://mastodon.social/users/Sommer",
		    "https://mastodon.social/users/ironfroggy",
		    "https://mastodon.social/users/fribbledom",
		    "https://mastodon.social/users/ControversyRecords",
		    "https://mastodon.social/users/envgen",
		    "https://mastodon.social/users/shesgabrielle",
		    "https://mastodon.social/users/ottaross",
		    "https://bsd.network/users/phessler",
		    "https://toot.cat/users/forktogether",
		    "https://chaos.social/users/dmitri",
		    "https://cybre.space/users/qwazix",
		    "https://mastodon.social/users/dheadshot"
		  ]
		}
	*/
	rootContext := []interface{}{
		"https://www.w3.org/ns/activitystreams",
		map[string]interface{}{
			"items": map[string]interface{}{
				"@id":   "as:items",
				"@type": "@id",
			},
		},
	}
	document := map[string]interface{}{
		"@context":   rootContext,
		"id":         actor.FollowersPage(page),
		"type":       "OrderedCollectionPage",
		"totalItems": len(followers),
		// next
		"partOf": actor.Followers(),
	}
	if len(followers) > 0 {
		document["orderedItems"] = followers
	}
	if len(followers) == 20 {
		document["next"] = actor.FollowersPage(page + 1)
	}
	if page > 1 {
		document["last"] = actor.FollowersPage(page - 1)
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
	if !matchContentType(c) {
		c.AbortWithStatus(http.StatusExpectationFailed)
		return
	}

	user := c.Param("user")

	ok, err := model.ActorLookup(h.DB, user, h.Domain)
	if err != nil {
		h.Logger.Error("failed looking up user", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !ok {
		h.Logger.Error("user not found", zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	following, err := model.FollowingLookup(h.DB, string(model.NewActorID(user, h.Domain)))

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Pragma", "no-cache")
	c.JSON(200, createFollowingResponse(user, h.Domain, following))
}
