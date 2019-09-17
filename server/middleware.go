package server

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/ngerakines/tavern/model"
	"go.uber.org/zap"
	"net/http"
)

func MatchContentTypeMiddleware(i *gin.Context) {
	if !matchContentType(i) {
		i.AbortWithStatus(http.StatusExpectationFailed)
		return
	}
	i.Next()
}

func UserExistsMiddleware(logger *zap.Logger, db *gorm.DB, domain string) func(*gin.Context) {
	return func(c *gin.Context) {
		user := c.Param("user")

		ok, err := model.ActorLookup(db, user, domain)
		if err != nil {
			logger.Error("failed looking up user", zap.Error(err), zap.String("user", user), zap.String("domain", domain))
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if !ok {
			logger.Error("user not found", zap.String("user", user), zap.String("domain", domain))
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Next()
	}
}
