package handler

import (
	"github.com/LJJsde/lrpc/center/configs"
	"github.com/LJJsde/lrpc/center/model"
	"github.com/LJJsde/lrpc/center/pkg/errcode"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func RegisterHandler(c *gin.Context) {
	log.Println("request api/register...")
	var req model.RequestRegister
	if e := c.ShouldBindJSON(&req); e != nil {
		log.Println("error:", e)
		err := errcode.ParamError
		c.JSON(http.StatusOK, gin.H{
			"code":    err.Code(),
			"message": err.Error(),
		})
		return
	}
	//bind instance
	instance := model.NewInstance(&req)
	if instance.Status != configs.StatusReceive && instance.Status != configs.StatusNotReceive {
		log.Println("register params status invalid")
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
		"data":    "",
	})
}
