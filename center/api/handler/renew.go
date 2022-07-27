package handler

import (
	"github.com/LJJsde/lrpc/center/model"
	"github.com/LJJsde/lrpc/center/pkg/errcode"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func RenewHandler(c *gin.Context) {
	log.Println("request api/renew...")
	var req model.RequestRenew
	if e := c.ShouldBindJSON(&req); e != nil {
		log.Println("error:", e)
		err := errcode.ParamError
		c.JSON(http.StatusOK, gin.H{
			"code":    err.Code(),
			"message": err.Error(),
		})
		return
	}

}
