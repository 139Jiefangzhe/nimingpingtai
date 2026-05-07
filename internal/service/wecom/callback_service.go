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

package wecom

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/apache/answer/internal/base/reason"
	"github.com/go-resty/resty/v2"
	"github.com/segmentfault/pacman/errors"
	"github.com/segmentfault/pacman/log"
)

type callbackEnvelope struct {
	XMLName xml.Name `xml:"xml"`
	Encrypt string   `xml:"Encrypt"`
}

type callbackMessage struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	MsgType      string   `xml:"MsgType"`
	Event        string   `xml:"Event"`
	ChangeType   string   `xml:"ChangeType"`
	InfoType     string   `xml:"InfoType"`
	AgentID      string   `xml:"AgentID"`
	UserID       string   `xml:"UserID"`
}

func (s *Service) VerifyURL(msgSignature, timestamp, nonce, echoStr string) (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	if err := validateCallbackConfig(cfg); err != nil {
		return "", err
	}
	if strings.TrimSpace(echoStr) == "" {
		return "", errors.BadRequest(reason.RequestFormatError)
	}
	if !verifyCallbackSignature(cfg.CallbackToken, timestamp, nonce, echoStr, msgSignature) {
		return "", errors.BadRequest(reason.UnauthorizedError)
	}

	plain, err := decryptCallbackPayload(cfg.CallbackAESKey, echoStr, cfg.CorpID)
	if err != nil {
		return "", errors.BadRequest(reason.UnauthorizedError).WithError(err).WithStack()
	}
	return string(plain), nil
}

func (s *Service) HandleEventCallback(ctx context.Context, msgSignature, timestamp, nonce string, body []byte) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if err := validateCallbackConfig(cfg); err != nil {
		return err
	}

	msg, err := parseCallbackMessage(cfg, msgSignature, timestamp, nonce, body)
	if err != nil {
		return err
	}

	log.Infof("wecom callback received msg_type=%s event=%s change_type=%s info_type=%s from=%s agent_id=%s",
		msg.MsgType, msg.Event, msg.ChangeType, msg.InfoType, msg.FromUserName, msg.AgentID)

	if msg.MsgType == "event" && msg.Event == "change_contact" {
		userID := strings.TrimSpace(msg.UserID)
		if userID == "" {
			userID = strings.TrimSpace(msg.FromUserName)
		}
		switch msg.ChangeType {
		case "create_user":
			log.Infof("wecom event: create_user user_id=%s", userID)
		case "update_user":
			log.Infof("wecom event: update_user user_id=%s", userID)
		case "delete_user":
			log.Infof("wecom event: delete_user user_id=%s, deactivating anonymous identity", userID)
			if err := s.deactivateUserIdentity(ctx, cfg, userID); err != nil {
				log.Errorf("failed to deactivate identity for user %s: %v", userID, err)
			}
		}
	}

	return nil
}

func parseCallbackMessage(cfg *config, msgSignature, timestamp, nonce string, body []byte) (*callbackMessage, error) {
	envelope := &callbackEnvelope{}
	if err := xml.Unmarshal(body, envelope); err != nil {
		return nil, errors.BadRequest(reason.RequestFormatError).WithError(err).WithStack()
	}
	if strings.TrimSpace(envelope.Encrypt) == "" {
		return nil, errors.BadRequest(reason.RequestFormatError)
	}
	if !verifyCallbackSignature(cfg.CallbackToken, timestamp, nonce, envelope.Encrypt, msgSignature) {
		return nil, errors.BadRequest(reason.UnauthorizedError)
	}

	plain, err := decryptCallbackPayload(cfg.CallbackAESKey, envelope.Encrypt, cfg.CorpID)
	if err != nil {
		return nil, errors.BadRequest(reason.UnauthorizedError).WithError(err).WithStack()
	}

	msg := &callbackMessage{}
	if err := xml.Unmarshal(plain, msg); err != nil {
		return nil, errors.BadRequest(reason.RequestFormatError).WithError(err).WithStack()
	}
	return msg, nil
}

func validateCallbackConfig(cfg *config) error {
	if cfg.CallbackToken == "" || cfg.CallbackAESKey == "" || cfg.CorpID == "" {
		return errors.BadRequest(reason.ReadConfigFailed)
	}
	return nil
}

func verifyCallbackSignature(token, timestamp, nonce, encrypted, msgSignature string) bool {
	parts := []string{token, timestamp, nonce, encrypted}
	sort.Strings(parts)
	sum := sha1.Sum([]byte(strings.Join(parts, "")))
	return hex.EncodeToString(sum[:]) == msgSignature
}

func decryptCallbackPayload(aesKey, encrypted, receiveID string) ([]byte, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(aesKey + "=")
	if err != nil {
		return nil, fmt.Errorf("decode aes key: %w", err)
	}
	if len(decodedKey) != 32 {
		return nil, fmt.Errorf("invalid aes key length: %d", len(decodedKey))
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decode encrypted payload: %w", err)
	}
	if len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("invalid encrypted payload length")
	}

	block, err := aes.NewCipher(decodedKey)
	if err != nil {
		return nil, fmt.Errorf("new aes cipher: %w", err)
	}

	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, decodedKey[:aes.BlockSize]).CryptBlocks(plain, ciphertext)

	plain, err = pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return nil, err
	}
	if len(plain) < 20 {
		return nil, fmt.Errorf("decrypted payload too short")
	}

	messageLength := int(binary.BigEndian.Uint32(plain[16:20]))
	if len(plain) < 20+messageLength {
		return nil, fmt.Errorf("invalid message length")
	}

	message := plain[20 : 20+messageLength]
	actualReceiveID := string(plain[20+messageLength:])
	if receiveID != "" && actualReceiveID != receiveID {
		return nil, fmt.Errorf("unexpected receive id: %s", actualReceiveID)
	}
	return message, nil
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("invalid pkcs7 data length")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize || padding > len(data) {
		return nil, fmt.Errorf("invalid pkcs7 padding")
	}
	for _, b := range data[len(data)-padding:] {
		if int(b) != padding {
			return nil, fmt.Errorf("invalid pkcs7 padding content")
		}
	}
	return data[:len(data)-padding], nil
}

func (s *Service) deactivateUserIdentity(ctx context.Context, cfg *config, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return nil
	}
	httpClient := resty.New().SetTimeout(10 * time.Second)
	_, err := httpClient.R().
		SetContext(ctx).
		SetHeader("X-Vault-Token", cfg.VaultSharedToken).
		SetBody(map[string]string{
			"corp_id": cfg.CorpID,
			"user_id": userID,
			"status":  "inactive",
		}).
		Post(strings.TrimRight(cfg.VaultBaseURL, "/") + "/internal/identity/update-status")
	return err
}

func (s *Service) makeAnonSubjectIDFromCorpUser(corpID, userID string) string {
	mac := hmac.New(sha256.New, []byte(corpID+":vault"))
	_, _ = mac.Write([]byte(corpID + ":" + userID))
	return hex.EncodeToString(mac.Sum(nil))[:24]
}
