package handler

import (
	"github.com/LJJsde/lrpc/center/model"
	"github.com/LJJsde/lrpc/center/pkg/errcode"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func FetchHandler(c *gin.Context) {
	log.Println("request api/fetch...")
	var req model.RequestFetch
	if e := c.ShouldBindJSON(&req); e != nil {
		err := errcode.ParamError
		c.JSON(http.StatusOK, gin.H{
			"code":    err.Code(),
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "",
	})
}
