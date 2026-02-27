package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"qq-farm-bot/internal/config"
	"qq-farm-bot/internal/model"
	"qq-farm-bot/internal/store"
)

type loginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerReq struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6"`
}

func RegisterRoutes(r *gin.RouterGroup, cfg *config.Config, s *store.Store) {
	// POST /auth/register - Open registration
	r.POST("/register", func(c *gin.Context) {
		var req registerReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: username (3-32 chars) and password (6+ chars) required"})
			return
		}

		// Check if username already exists
		exists, err := s.UserExists(req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
			return
		}

		// Hash password
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "password hashing failed"})
			return
		}

		// Check if this is the first user (make admin)
		hasUsers, err := s.HasAnyUser()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}

		user := &model.User{
			Username:     req.Username,
			PasswordHash: string(hash),
			IsAdmin:      !hasUsers, // First user becomes admin
		}

		if err := s.CreateUser(user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}

		// Generate token for auto-login
		token, err := GenerateToken(cfg.JWTSecret, user.ID, user.Username, user.IsAdmin)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"is_admin": user.IsAdmin,
			},
		})
	})

	// POST /auth/login
	r.POST("/login", func(c *gin.Context) {
		var req loginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// Try database user first
		user, err := s.GetUserByUsername(req.Username)
		if err == nil {
			// Verify password
			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}

			token, err := GenerateToken(cfg.JWTSecret, user.ID, user.Username, user.IsAdmin)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"token": token,
				"user": gin.H{
					"id":       user.ID,
					"username": user.Username,
					"is_admin": user.IsAdmin,
				},
			})
			return
		}

		// Fallback to config admin (for backwards compatibility)
		if req.Username == cfg.AdminUser && req.Password == cfg.AdminPass {
			// Create admin user in database if not exists
			hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			adminUser := &model.User{
				Username:     cfg.AdminUser,
				PasswordHash: string(hash),
				IsAdmin:      true,
			}
			// Try to create, ignore if exists
			if err := s.CreateUser(adminUser); err == nil {
				user = adminUser
			} else {
				// User might already exist, fetch it
				user, _ = s.GetUserByUsername(cfg.AdminUser)
			}

			if user == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get/create admin user"})
				return
			}

			token, err := GenerateToken(cfg.JWTSecret, user.ID, user.Username, user.IsAdmin)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"token": token,
				"user": gin.H{
					"id":       user.ID,
					"username": user.Username,
					"is_admin": user.IsAdmin,
				},
			})
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
	})
}
