package api

import (
	"github.com/gin-gonic/gin"
	db "github.com/xiusl/bank/db/sqlc"
)

// Server http 服务
type Server struct {
	store  *db.Store
	router *gin.Engine
}

// NewServer 创建一个新的服务，并设置路由
func NewServer(store *db.Store) *Server {
	server := &Server{
		store: store,
	}
	router := gin.Default()

	// 设置路由
	router.POST("/accounts", server.createAccount)
	router.GET("/accounts/:id", server.getAccount)
	router.GET("/accounts", server.listAccount)

	server.router = router

	return server
}

// Start 开启服务器，address 监听的地址
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

// 格式化错误信息
func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
