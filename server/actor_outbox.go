package server

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/ngerakines/tavern/model"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strconv"
)

func (h ActorHandler) OutboxHandler(c *gin.Context) {
	user := c.Param("user")

	actorID, err := model.ActorUUID(h.DB, user, h.Domain)
	if err != nil {
		h.Logger.Error("unable to get actor", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if c.Query("page") == "" {
		h.OutboxIndexHandler(c, actorID)
		return
	}
	h.OutboxPageHandler(c, actorID)
}

func (h ActorHandler) OutboxIndexHandler(c *gin.Context, actorID uuid.UUID) {
	user := c.Param("user")
	actor := model.NewActorID(user, h.Domain)

	count, err := model.PublicActorActivityCount(h.DB, actorID)
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
		"id":         actor.Outbox(),
		"totalItems": count,
	}
	if count > 0 {
		document["first"] = actor.OutboxPage(1)
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

func (h ActorHandler) OutboxPageHandler(c *gin.Context, actorID uuid.UUID) {
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

	activity, err := model.PublicActorActivity(h.DB, actorID, page, activityPerPage)
	if err != nil {
		h.Logger.Error("unable to get followers", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	var activityIDs []string
	for _, a := range activity {
		activityIDs = append(activityIDs, a.ActivityID.String())
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
		"id":         actor.OutboxPage(page),
		"type":       "OrderedCollectionPage",
		"totalItems": len(activityIDs),
		"partOf":     actor.Outbox(),
	}
	if len(activityIDs) > 0 {
		document["orderedItems"] = activityIDs
	}
	if len(activityIDs) == followersPerPage {
		document["next"] = actor.OutboxPage(page + 1)
	}
	if page > 1 {
		document["prev"] = actor.OutboxPage(page - 1)
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

func (h ActorHandler) OutboxSubmitHandler(c *gin.Context) {
	user := c.Param("user")
	actor := model.NewActorID(user, h.Domain)
	_, err := model.ActorUUID(h.DB, user, h.Domain)
	if err != nil {
		h.Logger.Error("unable to get actor", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var body []byte
	var document map[string]interface{}
	if body, err = ioutil.ReadAll(c.Request.Body); err != nil {
		h.Logger.Error("unable to read request body",
			zap.Error(err),
			zap.String("user", user),
			zap.String("domain", h.Domain),
			zap.String("content_type", c.GetHeader("Content-Type")))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if err = json.Unmarshal(body, &document); err != nil {
		h.Logger.Error("unable to parse JSON",
			zap.Error(err),
			zap.String("user", user),
			zap.String("domain", h.Domain),
			zap.String("content_type", c.GetHeader("Content-Type")))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	documentContext := map[string]interface{}{
		"@context": []interface{}{
			"https://www.w3.org/ns/activitystreams",
		},
	}

	result, err := compactJSONLD(document, documentContext)
	if err != nil {
		h.Logger.Error("unable to create response", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if !validateOutboxPayload(result) {
		h.Logger.Error("payload did not validate", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain), zap.Any("document", result))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	activityID, _ := uuid.NewV4()
	objectID, _ := uuid.NewV4()

	obj, err := objectFromPayload(result)
	if err != nil {
		h.Logger.Error("unable to create object", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	fullActivityID := fmt.Sprintf("https://%s/activity/%s", h.Domain, activityID.String())
	result["id"] = fullActivityID
	fullObjectID := fmt.Sprintf("https://%s/object/%s", h.Domain, objectID.String())
	obj["id"] = fullObjectID
	obj["attributedTo"] = string(actor)

	//err = model.RunTransactionWithOptions(h.DB, func(tx *gorm.DB) error {
	//	activityRec := model.Activity{
	//		ID:        activityID,
	//		ObjectID:  fullActivityID,
	//		Payload:   result,
	//		CreatedAt: time.Time{},
	//		UpdatedAt: time.Time{},
	//	}
	//	if err = tx.Save(&activityRec).Error; err != nil {
	//		return err
	//	}
	//	actorActivityRec := &model.ActorActivity{
	//		ActorID:    actorID,
	//		ActivityID: activityID,
	//		Public:     true,
	//	}
	//	if err = tx.Save(&actorActivityRec).Error; err != nil {
	//		return err
	//	}
	//	objectRec := &model.Object{
	//		ID:       objectID,
	//		ObjectID: fullObjectID,
	//		Payload:  obj,
	//	}
	//	if err = tx.Save(&objectRec).Error; err != nil {
	//		return err
	//	}
	//	return nil
	//})
	//if err != nil {
	//	h.Logger.Error("unable to store activity", zap.Error(err), zap.String("user", user), zap.String("domain", h.Domain))
	//	c.AbortWithStatus(http.StatusInternalServerError)
	//	return
	//}

	result["object"] = obj

	WriteJSONLD(c, result)
}

func validateOutboxPayload(document map[string]interface{}) bool {
	var ok bool
	var t string
	var published string
	var content string
	if t, ok = model.JSONString(document, "type"); !ok {
		return false
	}
	if !model.StringsContainsString([]string{"Note"}, t) {
		return false
	}

	if published, ok = model.JSONString(document, "published"); !ok {
		return false
	}
	if len(published) < 1 {
		return false
	}

	if content, ok = model.JSONString(document, "content"); !ok {
		return false
	}
	if len(content) < 1 {
		return false
	}
	return true
}

func objectFromPayload(document map[string]interface{}) (map[string]interface{}, error) {
	var obj map[string]interface{}
	var ok bool

	obj, ok = model.JSONMap(document, "object")
	if !ok || obj == nil {
		obj = make(map[string]interface{})
	}

	obj["type"] = document["type"]
	obj["content"] = document["content"]
	obj["published"] = document["published"]
	if _, ok = document["to"]; ok {
		obj["to"] = document["to"]
	}
	if _, ok = document["cc"]; ok {
		obj["cc"] = document["cc"]
	}
	if _, ok = document["bcc"]; ok {
		obj["bcc"] = document["bcc"]
	}

	return obj, nil
}
