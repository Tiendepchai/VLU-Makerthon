package middleware

import (
    "makerthon/models"
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

func OptionalAuth(db *gorm.DB) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        userIDStr, err := ctx.Cookie("user_id")
        if err != nil {
            ctx.Next()
            return
        }

        userID, err := strconv.Atoi(userIDStr)
        if err != nil {
            ctx.SetCookie("user_id", "", -1, "/", "", false, true)
            ctx.Next()
            return
        }

        var user models.User
        if err := db.First(&user, userID).Error; err != nil {
            ctx.SetCookie("user_id", "", -1, "/", "", false, true)
            ctx.Next()
            return
        }

        ctx.Set("user", user)
        ctx.Next()
    }
}

func RequireAuth(db *gorm.DB) gin.HandlerFunc {
    return func(ctx *gin.Context) {
        userIdStr, err := ctx.Cookie("user_id")
        if err != nil {
            ctx.Redirect(http.StatusFound, "/auth/login")
            ctx.Abort()
            return
        }

        userId, err := strconv.Atoi(userIdStr)
        if err != nil {
            ctx.SetCookie("user_id", "", -1, "/", "", false, true)
            ctx.Redirect(http.StatusFound, "/auth/login")
            ctx.Abort()
            return
        }

        var user models.User
        if err := db.First(&user, userId).Error; err != nil {
            ctx.SetCookie("user_id", "", -1, "/", "", false, true)
            ctx.Redirect(http.StatusFound, "/auth/login")
            ctx.Abort()
            return
        }

        if (!user.IsActive || (!user.EmailVerified && user.OAuthProvider == "")) {
            ctx.SetCookie("user_id", "", -1, "/", "", false, true)
            ctx.Redirect(http.StatusFound, "/auth/verify?email="+user.Email)
            ctx.Abort()
            return
        }

        ctx.Set("user", user)
        ctx.Next()
    }
}