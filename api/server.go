package api

import (
    "fmt"

    "github.com/gin-gonic/gin"
    "github.com/gin-gonic/gin/binding"
    "github.com/go-playground/validator/v10"
    db "github.com/xiusl/bank/db/sqlc"
    "github.com/xiusl/bank/token"
    "github.com/xiusl/bank/util"
)

// Server http 服务
type Server struct {
    store      db.Store
    router     *gin.Engine
    config     util.Config
    tokenMaker token.Maker
}

// NewServer 创建一个新的服务，并设置路由
func NewServer(config util.Config, store db.Store) (*Server, error) {
    tokenMaker, err := token.NewPasetoMaker(config.TokenSymmertricKey)
    if err != nil {
        return nil, fmt.Errorf("cannot create token maker: %w", err)
    }
    server := &Server{
        config:     config,
        store:      store,
        tokenMaker: tokenMaker,
    }

    if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
        v.RegisterValidation("currency", validCurrency)
    }

    server.setupRouter()

    return server, nil
}

func (server *Server) setupRouter() {
    router := gin.Default()

    // 设置路由

    router.POST("/users", server.createUser)
    router.POST("/users/login", server.loginUser)

    authRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker))

    authRoutes.POST("/accounts", server.createAccount)
    authRoutes.GET("/accounts/:id", server.getAccount)
    authRoutes.GET("/accounts", server.listAccount)
    authRoutes.POST("/transfers", server.createTransfer)

    server.router = router
}

// Start 开启服务器，address 监听的地址
func (server *Server) Start(address string) error {
    return server.router.Run(address)
}

// 格式化错误信息
func errorResponse(err error) gin.H {
    return gin.H{"error": err.Error()}
}
