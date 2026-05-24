package middelware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GinContextErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process request
		c.Next()

		// Only handle errors if response status is OK (no error response written yet)
		if len(c.Errors) > 0 && c.Writer.Status() == http.StatusOK {
			err := c.Errors.Last()

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Err.Error(),
			})
			return
		}

		// Log errors if response was already written with error status
		if len(c.Errors) > 0 && c.Writer.Status() != http.StatusOK {
			err := c.Errors.Last()
			log.Printf("Gin error (status %d): %v", c.Writer.Status(), err.Err)
		}
	}
}
