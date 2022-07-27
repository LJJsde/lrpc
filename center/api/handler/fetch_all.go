package handler

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func FetchAllHandler(c *gin.Context) {
	log.Println("request api/fetchall...")
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "",
	})
}
