// Author: Kazooki123, StarloExoliz

/*
** Copyright 2024 Kazooki123
**
** Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
** documentation files (the “Software”), to deal in the Software without restriction, including without
** limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
** of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following
** conditions:
**
** The above copyright notice and this permission notice shall be included in all copies or substantial
** portions of the Software.
**
** THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT
** LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
** IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
** LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH
** THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
**
**/


/*
**
** This mimics on the functions of LunarDB
** As executing the "lunar.exe" file itself is very tricky
** So instead we code it in main.go
**
**/


package main

import (
	"log"
	"crypto/rand"
    "encoding/base64"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type APIKeyManager struct {
    keys map[string]bool
    mu   sync.RWMutex
}

type LunarDB struct {
	data map[string]string
	mu   sync.RWMutex
}

func NewAPIKeyManager() *APIKeyManager {
    return &APIKeyManager{
        keys: make(map[string]bool),
    }
}

func (m *APIKeyManager) GenerateKey() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}

func (m *APIKeyManager) AddKey(key string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.keys[key] = true
}

func (m *APIKeyManager) ValidateKey(key string) bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.keys[key]
}

func APIKeyMiddleware(keyManager *APIKeyManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := c.GetHeader("X-API-Key")
        if key == "" || !keyManager.ValidateKey(key) {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
            return
        }
        c.Next()
    }
}

func NewLunarDB() *LunarDB {
	return &LunarDB{
		data: make(map[string]string),
	}
}

func (db *LunarDB) Set(key, value string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.data[key] = value
}

func (db *LunarDB) Get(key string) (string, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	value, exists := db.data[key]
	return value, exists
}

func (db *LunarDB) Del(key string) bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, exists := db.data[key]
	if exists {
		delete(db.data, key)
	}
	return exists
}

func (db *LunarDB) Keys() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	keys := make([]string, 0, len(db.data))
	for k := range db.data {
		keys = append(keys, k)
	}
	return keys
}

var (
	db *LunarDB
	keyManager *APIKeyManager
)

// Main
func main() {
	db = NewLunarDB()
	keyManager = NewAPIKeyManager()

	// Generate and add an initial API key
    initialKey := keyManager.GenerateKey()
    keyManager.AddKey(initialKey)
    log.Printf("Initial API Key: %s", initialKey)

	r := gin.Default()

	// Setup routes
	setupRoutes(r)

	// Start server
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupRoutes(r *gin.Engine) {
	// Public routes:
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to LunarDB API",
		})
	})

	r.GET("/health", healthHandler)

	// Protected routes (w/ api):
	v1 := r.Group("/api/v1")
	v1.Use(APIKeyMiddleware(keyManager))
	{
		v1.POST("/set", setHandler)
		v1.GET("/get/:key", getHandler)
		v1.DELETE("/del/:key", delHandler)
		v1.GET("/keys", keysHandler)
		v1.POST("/query", queryHandler)
		v1.POST("/schema", schemaHandler)
	}
}

func setHandler(c *gin.Context) {
	var request struct {
		Key   string `json:"key" binding:"required"`
		Value string `json:"value" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.Set(request.Key, request.Value)
	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}

func getHandler(c *gin.Context) {
	key := c.Param("key")
	value, exists := db.Get(key)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Key not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": value})
}

func delHandler(c *gin.Context) {
	key := c.Param("key")
	existed := db.Del(key)
	if !existed {
		c.JSON(http.StatusNotFound, gin.H{"error": "Key not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": "OK"})
}

func keysHandler(c *gin.Context) {
	keys := db.Keys()
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func queryHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Query endpoint not implemented yet"})
}

func schemaHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Schema endpoint not implemented yet"})
}
