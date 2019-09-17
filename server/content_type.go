package server

import (
	"github.com/gin-gonic/gin"
	"github.com/piprate/json-gold/ld"
)

func compactJSONLD(document, context map[string]interface{}) (map[string]interface{}, error) {
	proc := ld.NewJsonLdProcessor()
	options := ld.NewJsonLdOptions("https://www.w3.org/ns/activitystreams")
	options.ProcessingMode = ld.JsonLd_1_1
	return proc.Compact(document, context, options)
}

func WriteJSONLD(c *gin.Context, data map[string]interface{}) {
	c.Writer.Header().Set("Content-Type", "application/activity+json")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Pragma", "no-cache")
	c.JSON(200, data)
}

func matchContentType(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	if accept == "application/activity+json" {
		return true
	} else if accept == `application/ld+json; profile="https://www.w3.org/ns/activitystreams"` {
		return true
	}

	contentType := c.GetHeader("Content-Type")
	if contentType == "application/activity+json" {
		return true
	} else if contentType == `application/ld+json; profile="https://www.w3.org/ns/activitystreams"` {
		return true
	}

	return false
}

func jsonldContextFollowers() interface{} {
	return []interface{}{
		"https://www.w3.org/ns/activitystreams",
		map[string]interface{}{
			"schema": "http://schema.org#",
			"items":  "as:items",
		},
	}
}

func jsonldContextFollowering() interface{} {
	return []interface{}{
		"https://www.w3.org/ns/activitystreams",
		map[string]interface{}{
			"items": map[string]interface{}{
				"@id":   "as:items",
				"@type": "@id",
			},
		},
	}
}
