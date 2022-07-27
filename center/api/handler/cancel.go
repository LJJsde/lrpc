package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/LJJsde/lrpc/center/configs"
	"github.com/LJJsde/lrpc/center/model"
	"github.com/LJJsde/lrpc/center/pkg/errcode"
	"log"
	"net/http"
)

func CancelHandler(c *gin.Context) {
	log.Println("request api/cancel...")
	var req model.RequestCancel
	if e := c.ShouldBindJSON(&req); e != nil {
		err := errcode.ParamError
		c.JSON(http.StatusOK, gin.H{
			"code":    err.Code(),
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    configs.StatusOK,
		"message": "",
	})
}
