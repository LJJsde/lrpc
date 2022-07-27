package api

import (
	"github.com/LJJsde/lrpc/center/api/handler"
	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	router := gin.Default()
	router.POST("api/register", handler.RegisterHandler)
	router.POST("api/fetch", handler.FetchHandler)
	router.POST("api/renew", handler.RenewHandler)
	router.POST("api/cancel", handler.CancelHandler)
	router.POST("api/fetchall", handler.FetchAllHandler)
	router.POST("api/nodes", handler.NodesHandler)
	return router
}
