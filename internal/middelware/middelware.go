package middelware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GinContextErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			// Get the most relevant (last) error thrown
			err := c.Errors.Last()

			// Check if a specific HTTP status was given, else default to 500
			status := c.Writer.Status()
			if status == http.StatusOK {
				status = http.StatusInternalServerError
			}

			c.JSON(status, gin.H{
				"error": err.Err.Error(),
			})
		}
	}
}
