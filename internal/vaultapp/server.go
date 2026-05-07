/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package vaultapp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/apache/answer/internal/schema"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

type IdentityMapping struct {
	ID            int64     `xorm:"not null pk autoincr BIGINT(20) id"`
	CreatedAt     time.Time `xorm:"created TIMESTAMP created_at"`
	UpdatedAt     time.Time `xorm:"updated TIMESTAMP updated_at"`
	CorpID        string    `xorm:"not null default '' VARCHAR(128) INDEX(corp_user) corp_id"`
	UserID        string    `xorm:"not null default '' VARCHAR(128) INDEX(corp_user) user_id"`
	AnonSubjectID string    `xorm:"not null default '' VARCHAR(64) unique anon_subject_id"`
	Status        string    `xorm:"not null default 'active' VARCHAR(20) status"`
	EncryptedBlob string    `xorm:"TEXT encrypted_blob"`
}

func (IdentityMapping) TableName() string {
	return "identity_mapping"
}

type AuditRevealLog struct {
	ID              int64     `xorm:"not null pk autoincr BIGINT(20) id"`
	CreatedAt       time.Time `xorm:"created TIMESTAMP created_at"`
	RequesterUserID string    `xorm:"not null default '' VARCHAR(128) requester_user_id"`
	AnonSubjectID   string    `xorm:"not null default '' VARCHAR(64) INDEX anon_subject_id"`
	Action          string    `xorm:"not null default '' VARCHAR(32) action"`
	Reason          string    `xorm:"TEXT reason"`
	Metadata        string    `xorm:"TEXT metadata"`
}

func (AuditRevealLog) TableName() string {
	return "audit_reveal_log"
}

type identityBlob struct {
	CorpID      string `json:"corp_id"`
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Mobile      string `json:"mobile"`
	Department  string `json:"department"`
	Position    string `json:"position"`
	Avatar      string `json:"avatar"`
	Status      string `json:"status"`
}

type Config struct {
	ListenAddr  string
	DBDriver    string
	DBDSN       string
	SharedToken string
	Secret      string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		ListenAddr:  envOrDefault("VAULT_LISTEN_ADDR", ":8091"),
		DBDriver:    envOrDefault("VAULT_DB_DRIVER", "sqlite"),
		DBDSN:       envOrDefault("VAULT_DB_DSN", "/tmp/vault.db"),
		SharedToken: os.Getenv("VAULT_SHARED_TOKEN"),
		Secret:      os.Getenv("VAULT_SECRET"),
	}
	if cfg.SharedToken == "" || cfg.Secret == "" {
		return nil, fmt.Errorf("VAULT_SHARED_TOKEN and VAULT_SECRET must be set")
	}
	return cfg, nil
}

func NewServer(cfg *Config) (*gin.Engine, func() error, error) {
	engine, err := xorm.NewEngine(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		return nil, nil, err
	}
	if err = engine.Sync(new(IdentityMapping), new(AuditRevealLog)); err != nil {
		return nil, nil, err
	}

	srv := &service{db: engine, cfg: cfg}
	r := gin.New()
	r.Use(gin.Recovery(), srv.authz())
	r.GET("/healthz", func(ctx *gin.Context) { ctx.String(http.StatusOK, "OK") })
	r.POST("/internal/identity/resolve", srv.resolve)
	r.POST("/internal/identity/status", srv.status)
	r.POST("/internal/identity/reveal", srv.reveal)
	r.POST("/internal/audit/log", srv.auditLog)
	return r, engine.Close, nil
}

type service struct {
	db  *xorm.Engine
	cfg *Config
}

func (s *service) authz() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.URL.Path == "/healthz" {
			ctx.Next()
			return
		}
		if ctx.GetHeader("X-Vault-Token") != s.cfg.SharedToken {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		ctx.Next()
	}
}

func (s *service) resolve(ctx *gin.Context) {
	req := &schema.VaultResolveRequest{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mapping := &IdentityMapping{}
	exist, err := s.db.Context(ctx).Where("corp_id = ?", req.CorpID).And("user_id = ?", req.UserID).Get(mapping)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !exist {
		anonSubjectID := s.makeAnonSubjectID(req.CorpID, req.UserID)
		blob := identityBlob{
			CorpID:      req.CorpID,
			UserID:      req.UserID,
			DisplayName: req.DisplayName,
			Email:       req.Email,
			Mobile:      req.Mobile,
			Department:  req.Department,
			Position:    req.Position,
			Avatar:      req.Avatar,
			Status:      "active",
		}
		encrypted, err := s.encrypt(blob)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		mapping = &IdentityMapping{
			CorpID:        req.CorpID,
			UserID:        req.UserID,
			AnonSubjectID: anonSubjectID,
			Status:        "active",
			EncryptedBlob: encrypted,
		}
		if _, err = s.db.Context(ctx).Insert(mapping); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	ctx.JSON(http.StatusOK, &schema.VaultResolveResponse{
		AnonSubjectID: mapping.AnonSubjectID,
		DisplayName:   makeAnonymousDisplayName(mapping.AnonSubjectID),
		AvatarSeed:    makeAvatarSeed(mapping.AnonSubjectID),
		Avatar:        "",
		Status:        mapping.Status,
	})
}

func (s *service) status(ctx *gin.Context) {
	req := &schema.VaultStatusRequest{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mapping := &IdentityMapping{}
	exist, err := s.db.Context(ctx).Where("anon_subject_id = ?", req.AnonSubjectID).Get(mapping)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exist {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	ctx.JSON(http.StatusOK, &schema.VaultStatusResponse{
		AnonSubjectID: mapping.AnonSubjectID,
		Status:        mapping.Status,
	})
}

func (s *service) reveal(ctx *gin.Context) {
	req := &schema.VaultRevealRequest{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mapping := &IdentityMapping{}
	exist, err := s.db.Context(ctx).Where("anon_subject_id = ?", req.AnonSubjectID).Get(mapping)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !exist {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	blob, err := s.decrypt(mapping.EncryptedBlob)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_, _ = s.db.Context(ctx).Insert(&AuditRevealLog{
		RequesterUserID: req.RequesterUserID,
		AnonSubjectID:   req.AnonSubjectID,
		Action:          "reveal",
		Reason:          req.Reason,
	})
	ctx.JSON(http.StatusOK, &schema.VaultRevealResponse{
		AnonSubjectID: mapping.AnonSubjectID,
		CorpID:        blob.CorpID,
		UserID:        blob.UserID,
		DisplayName:   blob.DisplayName,
		Email:         blob.Email,
		Mobile:        blob.Mobile,
		Status:        mapping.Status,
	})
}

func (s *service) auditLog(ctx *gin.Context) {
	req := &schema.VaultAuditLogRequest{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := s.db.Context(ctx).Insert(&AuditRevealLog{
		RequesterUserID: req.RequesterUserID,
		AnonSubjectID:   req.AnonSubjectID,
		Action:          req.Action,
		Reason:          req.Reason,
		Metadata:        req.Metadata,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *service) makeAnonSubjectID(corpID, userID string) string {
	mac := hmac.New(sha256.New, []byte(s.cfg.Secret))
	_, _ = mac.Write([]byte(corpID + ":" + userID))
	return hex.EncodeToString(mac.Sum(nil))[:24]
}

func (s *service) encrypt(blob identityBlob) (string, error) {
	raw, err := json.Marshal(blob)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(hashSecret(s.cfg.Secret))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, raw, nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (s *service) decrypt(ciphertext string) (*identityBlob, error) {
	raw, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(hashSecret(s.cfg.Secret))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(raw) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, encrypted := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, err
	}
	resp := &identityBlob{}
	if err = json.Unmarshal(plain, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func hashSecret(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func makeAnonymousDisplayName(anonSubjectID string) string {
	suffix := anonSubjectID
	if len(suffix) > 4 {
		suffix = suffix[len(suffix)-4:]
	}
	return "匿名用户" + strings.ToUpper(suffix)
}

func makeAvatarSeed(anonSubjectID string) string {
	if len(anonSubjectID) > 8 {
		return anonSubjectID[len(anonSubjectID)-8:]
	}
	return anonSubjectID
}
