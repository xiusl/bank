package api

import (
    "errors"
    "fmt"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/xiusl/bank/token"
)

const (
    authorizationHeaderKey  = "authorization"
    authorizationTypeBearer = "bearer"
    authorizationPayloadKey = "authorization_payload"
)

func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        authorizationHeader := ctx.GetHeader(authorizationHeaderKey)
        if len(authorizationHeader) == 0 {
            err := errors.New("authorization header is not provided")
            ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
            return
        }

        fileds := strings.Fields(authorizationHeader)
        if len(fileds) < 2 {
            err := errors.New("authorization header is not provided")
            ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
            return
        }

        authorizationType := strings.ToLower(fileds[0])
        if authorizationType != authorizationTypeBearer {
            err := fmt.Errorf("unsuppored authorization type %s", authorizationType)
            ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
            return
        }

        accessToken := fileds[1]
        payload, err := tokenMaker.VerifyToken(accessToken)
        if err != nil {
            err := errors.New("token error")
            ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
            return
        }

        ctx.Set(authorizationPayloadKey, payload)
        ctx.Next()
    }
}
