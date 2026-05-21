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
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	userexternallogin "github.com/apache/answer/internal/service/user_external_login"
	"github.com/apache/answer/plugin"
	"github.com/go-resty/resty/v2"
	"github.com/segmentfault/pacman/errors"
	"github.com/segmentfault/pacman/log"
)

const providerSlug = "wecom-anonymous"

type Service struct {
	data                   *data.Data
	userCenterLoginService *userexternallogin.UserCenterLoginService
	userExternalLoginRepo  userexternallogin.UserExternalLoginRepo
	httpClient             *resty.Client
}

func NewWeComService(
	data *data.Data,
	userCenterLoginService *userexternallogin.UserCenterLoginService,
	userExternalLoginRepo userexternallogin.UserExternalLoginRepo,
) *Service {
	return &Service{
		data:                   data,
		userCenterLoginService: userCenterLoginService,
		userExternalLoginRepo:  userExternalLoginRepo,
		httpClient:             resty.New().SetTimeout(10 * time.Second),
	}
}

type config struct {
	CorpID           string
	AgentID          string
	AppSecret        string
	AppBaseURL       string
	DefaultReturnTo  string
	VaultBaseURL     string
	VaultSharedToken string
	CallbackToken    string
	CallbackAESKey   string
}

func (s *Service) GetAuthorizationURL(returnTo string) (*schema.WeComAuthStartResp, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	if returnTo == "" || !strings.HasPrefix(returnTo, "/") {
		returnTo = cfg.DefaultReturnTo
	}
	state := base64.RawURLEncoding.EncodeToString([]byte(returnTo))
	redirectURL := strings.TrimRight(cfg.AppBaseURL, "/") + "/answer/api/v1/wecom/auth/callback"
	query := url.Values{}
	query.Set("appid", cfg.CorpID)
	query.Set("redirect_uri", redirectURL)
	query.Set("response_type", "code")
	query.Set("scope", "snsapi_base")
	query.Set("state", state)
	if cfg.AgentID != "" {
		query.Set("agentid", cfg.AgentID)
	}
	authURL := "https://open.weixin.qq.com/connect/oauth2/authorize?" + query.Encode() + "#wechat_redirect"
	return &schema.WeComAuthStartResp{AuthorizationURL: authURL}, nil
}

func (s *Service) HandleAuthCallback(ctx context.Context, code, state string) (*schema.WeComAuthCallbackResp, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	wecomUser, err := s.fetchUserProfile(ctx, cfg, code)
	if err != nil {
		return nil, err
	}
	vaultResp, err := s.resolveIdentity(ctx, cfg, &schema.VaultResolveRequest{
		CorpID:      wecomUser.CorpID,
		UserID:      wecomUser.UserID,
		DisplayName: wecomUser.Name,
		Avatar:      wecomUser.Avatar,
		Email:       wecomUser.Email,
		Mobile:      wecomUser.Mobile,
		Department:  wecomUser.Department,
		Position:    wecomUser.Position,
	})
	if err != nil {
		return nil, err
	}

	loginResp, err := s.userCenterLoginService.ExternalLogin(ctx, anonymousUserCenter{}, &plugin.UserCenterBasicUserInfo{
		ExternalID:  vaultResp.AnonSubjectID,
		Username:    vaultResp.AnonSubjectID,
		DisplayName: vaultResp.DisplayName,
		Avatar:      vaultResp.Avatar,
	})
	if err != nil {
		return nil, err
	}
	if loginResp.ErrMsg != "" {
		return nil, errors.BadRequest(reason.UserAccessDenied)
	}

	if err := s.ensureAnonymousProfile(ctx, vaultResp); err != nil {
		return nil, err
	}

	returnTo := "/community"
	if state != "" {
		if decoded, decodeErr := base64.RawURLEncoding.DecodeString(state); decodeErr == nil && len(decoded) > 0 {
			candidate := string(decoded)
			if strings.HasPrefix(candidate, "/") {
				returnTo = candidate
			}
		}
	}

	return &schema.WeComAuthCallbackResp{
		AccessToken:   loginResp.AccessToken,
		RedirectURL:   strings.TrimRight(cfg.AppBaseURL, "/") + "/users/auth-landing?access_token=" + url.QueryEscape(loginResp.AccessToken) + "&return_to=" + url.QueryEscape(returnTo),
		AnonSubjectID: vaultResp.AnonSubjectID,
	}, nil
}

func (s *Service) RevealIdentity(ctx context.Context, requesterUserID string, req *schema.CommunityAuditRevealReq) (*schema.CommunityAuditRevealResp, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	vaultResp, err := s.revealIdentity(ctx, cfg, &schema.VaultRevealRequest{
		RequesterUserID: requesterUserID,
		AnonSubjectID:   req.AnonSubjectID,
		Reason:          req.Reason,
	})
	if err != nil {
		return nil, err
	}
	return &schema.CommunityAuditRevealResp{
		AnonSubjectID: vaultResp.AnonSubjectID,
		CorpID:        vaultResp.CorpID,
		UserID:        vaultResp.UserID,
		DisplayName:   vaultResp.DisplayName,
		Email:         vaultResp.Email,
		Mobile:        vaultResp.Mobile,
		Status:        vaultResp.Status,
	}, nil
}

func (s *Service) ensureAnonymousProfile(ctx context.Context, resolve *schema.VaultResolveResponse) error {
	link, exist, err := s.userExternalLoginRepo.GetByExternalID(ctx, providerSlug, resolve.AnonSubjectID)
	if err != nil || !exist {
		return err
	}
	profile := &entity.AnonymousProfile{UserID: link.UserID}
	has, err := s.data.DB.Context(ctx).Get(profile)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if has {
		profile.DisplayName = resolve.DisplayName
		profile.Avatar = resolve.Avatar
		profile.AvatarSeed = resolve.AvatarSeed
		profile.Status = resolve.Status
		_, err = s.data.DB.Context(ctx).ID(profile.UserID).Cols("display_name", "avatar", "avatar_seed", "status").Update(profile)
	} else {
		profile.AnonSubjectID = resolve.AnonSubjectID
		profile.DisplayName = resolve.DisplayName
		profile.Avatar = resolve.Avatar
		profile.AvatarSeed = resolve.AvatarSeed
		profile.Status = resolve.Status
		_, err = s.data.DB.Context(ctx).Insert(profile)
	}
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (s *Service) syncAnonymousProfileStatus(ctx context.Context, anonSubjectID, status string) error {
	if anonSubjectID == "" || status == "" {
		return nil
	}
	link, exist, err := s.userExternalLoginRepo.GetByExternalID(ctx, providerSlug, anonSubjectID)
	if err != nil || !exist {
		return err
	}
	profile := &entity.AnonymousProfile{UserID: link.UserID}
	has, err := s.data.DB.Context(ctx).Get(profile)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if !has {
		return nil
	}
	profile.Status = status
	_, err = s.data.DB.Context(ctx).ID(profile.UserID).Cols("status").Update(profile)
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (s *Service) resolveIdentity(ctx context.Context, cfg *config, req *schema.VaultResolveRequest) (*schema.VaultResolveResponse, error) {
	resp := &schema.VaultResolveResponse{}
	_, err := s.httpClient.R().
		SetContext(ctx).
		SetHeader("X-Vault-Token", cfg.VaultSharedToken).
		SetBody(req).
		SetResult(resp).
		Post(strings.TrimRight(cfg.VaultBaseURL, "/") + "/internal/identity/resolve")
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) updateIdentityStatus(ctx context.Context, cfg *config, req *schema.VaultUpdateStatusRequest) (*schema.VaultUpdateStatusResponse, error) {
	resp := &schema.VaultUpdateStatusResponse{}
	_, err := s.httpClient.R().
		SetContext(ctx).
		SetHeader("X-Vault-Token", cfg.VaultSharedToken).
		SetBody(req).
		SetResult(resp).
		Post(strings.TrimRight(cfg.VaultBaseURL, "/") + "/internal/identity/update-status")
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) revealIdentity(ctx context.Context, cfg *config, req *schema.VaultRevealRequest) (*schema.VaultRevealResponse, error) {
	resp := &schema.VaultRevealResponse{}
	_, err := s.httpClient.R().
		SetContext(ctx).
		SetHeader("X-Vault-Token", cfg.VaultSharedToken).
		SetBody(req).
		SetResult(resp).
		Post(strings.TrimRight(cfg.VaultBaseURL, "/") + "/internal/identity/reveal")
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type accessTokenResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
}

type authUserResponse struct {
	ErrCode        int    `json:"errcode"`
	ErrMsg         string `json:"errmsg"`
	UserID         string `json:"userid"`
	UserTicket     string `json:"user_ticket"`
	UserDocTicket  string `json:"user_doc_ticket"`
	OpenID         string `json:"openid"`
	ExternalUserID string `json:"external_userid"`
}

type userProfileResponse struct {
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
	UserID     string `json:"userid"`
	Name       string `json:"name"`
	Avatar     string `json:"avatar"`
	Email      string `json:"email"`
	Mobile     string `json:"mobile"`
	Position   string `json:"position"`
	Department []int  `json:"department"`
}

type linkedCorpUserProfileResponse struct {
	ErrCode  int    `json:"errcode"`
	ErrMsg   string `json:"errmsg"`
	UserInfo struct {
		UserID     string   `json:"userid"`
		CorpID     string   `json:"corpid"`
		Name       string   `json:"name"`
		Avatar     string   `json:"avatar"`
		Email      string   `json:"email"`
		Mobile     string   `json:"mobile"`
		Position   string   `json:"position"`
		Department []string `json:"department"`
	} `json:"user_info"`
}

type wecomUserProfile struct {
	CorpID     string
	UserID     string
	Name       string
	Avatar     string
	Email      string
	Mobile     string
	Position   string
	Department string
}

func (s *Service) fetchUserProfile(ctx context.Context, cfg *config, code string) (*wecomUserProfile, error) {
	if strings.TrimSpace(code) == "" {
		return nil, errors.BadRequest(reason.RequestFormatError)
	}
	accessToken, err := s.getAccessToken(ctx, cfg)
	if err != nil {
		return nil, err
	}

	authResp := &authUserResponse{}
	_, err = s.httpClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"access_token": accessToken,
			"code":         code,
		}).
		SetResult(authResp).
		Get("https://qyapi.weixin.qq.com/cgi-bin/auth/getuserinfo")
	if err != nil {
		return nil, err
	}
	log.Infof(
		"wecom auth/getuserinfo result errcode=%d errmsg=%s userid=%q openid_present=%t external_userid_present=%t user_ticket_present=%t",
		authResp.ErrCode,
		authResp.ErrMsg,
		authResp.UserID,
		authResp.OpenID != "",
		authResp.ExternalUserID != "",
		authResp.UserTicket != "",
	)
	if authResp.UserID == "" {
		log.Warnf(
			"wecom auth/getuserinfo missing userid errcode=%d errmsg=%s openid=%q external_userid=%q",
			authResp.ErrCode,
			authResp.ErrMsg,
			authResp.OpenID,
			authResp.ExternalUserID,
		)
		return nil, errors.BadRequest(reason.UnauthorizedError)
	}

	if isLinkedCorpUserID(authResp.UserID) {
		return s.fetchLinkedCorpUserProfileByUserID(ctx, cfg, accessToken, authResp.UserID)
	}

	return s.fetchUserProfileByUserID(ctx, cfg, accessToken, authResp.UserID)
}

func (s *Service) fetchUserProfileByUserID(ctx context.Context, cfg *config, accessToken, userID string) (*wecomUserProfile, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.BadRequest(reason.RequestFormatError)
	}
	if accessToken == "" {
		var err error
		accessToken, err = s.getAccessToken(ctx, cfg)
		if err != nil {
			return nil, err
		}
	}

	userResp := &userProfileResponse{}
	_, err := s.httpClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"access_token": accessToken,
			"userid":       userID,
		}).
		SetResult(userResp).
		Get("https://qyapi.weixin.qq.com/cgi-bin/user/get")
	if err != nil {
		return nil, err
	}
	log.Infof("wecom user/get result errcode=%d errmsg=%s userid=%q name=%q department_count=%d", userResp.ErrCode, userResp.ErrMsg, userResp.UserID, userResp.Name, len(userResp.Department))
	if userResp.UserID == "" {
		log.Warnf("wecom user/get returned empty userid errcode=%d errmsg=%s", userResp.ErrCode, userResp.ErrMsg)
		return nil, errors.BadRequest(reason.UserNotFound)
	}
	return &wecomUserProfile{
		CorpID:     cfg.CorpID,
		UserID:     userResp.UserID,
		Name:       userResp.Name,
		Avatar:     userResp.Avatar,
		Email:      userResp.Email,
		Mobile:     userResp.Mobile,
		Position:   userResp.Position,
		Department: joinDepartmentIntIDs(userResp.Department),
	}, nil
}

func (s *Service) fetchLinkedCorpUserProfileByUserID(ctx context.Context, cfg *config, accessToken, userID string) (*wecomUserProfile, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.BadRequest(reason.RequestFormatError)
	}
	if accessToken == "" {
		var err error
		accessToken, err = s.getAccessToken(ctx, cfg)
		if err != nil {
			return nil, err
		}
	}

	userResp := &linkedCorpUserProfileResponse{}
	_, err := s.httpClient.R().
		SetContext(ctx).
		SetQueryParam("access_token", accessToken).
		SetBody(map[string]string{"userid": userID}).
		SetResult(userResp).
		Post("https://qyapi.weixin.qq.com/cgi-bin/linkedcorp/user/get")
	if err != nil {
		return nil, err
	}
	log.Infof(
		"wecom linkedcorp/user/get result errcode=%d errmsg=%s userid=%q corpid=%q name=%q department_count=%d",
		userResp.ErrCode,
		userResp.ErrMsg,
		userResp.UserInfo.UserID,
		userResp.UserInfo.CorpID,
		userResp.UserInfo.Name,
		len(userResp.UserInfo.Department),
	)
	if userResp.UserInfo.UserID == "" || userResp.UserInfo.CorpID == "" {
		log.Warnf("wecom linkedcorp/user/get returned empty identity errcode=%d errmsg=%s", userResp.ErrCode, userResp.ErrMsg)
		return nil, errors.BadRequest(reason.UserNotFound)
	}
	return &wecomUserProfile{
		CorpID:     userResp.UserInfo.CorpID,
		UserID:     userResp.UserInfo.UserID,
		Name:       userResp.UserInfo.Name,
		Avatar:     userResp.UserInfo.Avatar,
		Email:      userResp.UserInfo.Email,
		Mobile:     userResp.UserInfo.Mobile,
		Position:   userResp.UserInfo.Position,
		Department: strings.Join(userResp.UserInfo.Department, ","),
	}, nil
}

func (s *Service) getAccessToken(ctx context.Context, cfg *config) (string, error) {
	tokenResp := &accessTokenResponse{}
	_, err := s.httpClient.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"corpid":     cfg.CorpID,
			"corpsecret": cfg.AppSecret,
		}).
		SetResult(tokenResp).
		Get("https://qyapi.weixin.qq.com/cgi-bin/gettoken")
	if err != nil {
		return "", err
	}
	log.Infof("wecom gettoken result errcode=%d errmsg=%s has_access_token=%t", tokenResp.ErrCode, tokenResp.ErrMsg, tokenResp.AccessToken != "")
	if tokenResp.AccessToken == "" {
		log.Warnf("wecom gettoken returned empty access token errcode=%d errmsg=%s", tokenResp.ErrCode, tokenResp.ErrMsg)
		return "", errors.BadRequest(reason.UnauthorizedError)
	}
	return tokenResp.AccessToken, nil
}

func (s *Service) NotifyReplyAuthor(ctx context.Context, anonSubjectID, postTitle, replyAuthor, replyExcerpt, postURL string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if strings.TrimSpace(anonSubjectID) == "" {
		return nil
	}

	lookupResp := &schema.VaultLookupResponse{}
	lookupHTTPResp, err := s.httpClient.R().
		SetContext(ctx).
		SetHeader("X-Vault-Token", cfg.VaultSharedToken).
		SetBody(map[string]string{"anon_subject_id": anonSubjectID}).
		SetResult(lookupResp).
		Post(strings.TrimRight(cfg.VaultBaseURL, "/") + "/internal/identity/lookup")
	if err != nil {
		return fmt.Errorf("vault lookup failed: %w", err)
	}
	if lookupHTTPResp.IsError() {
		return fmt.Errorf("vault lookup failed: status=%d body=%s", lookupHTTPResp.StatusCode(), strings.TrimSpace(lookupHTTPResp.String()))
	}
	if lookupResp.Status != "" && lookupResp.Status != "active" {
		return nil
	}
	if strings.TrimSpace(lookupResp.UserID) == "" {
		return fmt.Errorf("vault lookup returned empty user_id for anon_subject_id=%s", anonSubjectID)
	}

	agentID, err := strconv.Atoi(cfg.AgentID)
	if err != nil {
		return fmt.Errorf("invalid WECOM_AGENT_ID %q: %w", cfg.AgentID, err)
	}

	token, err := s.getAccessToken(ctx, cfg)
	if err != nil {
		return err
	}

	if strings.TrimSpace(postTitle) == "" {
		postTitle = "匿名社区新回复"
	}
	if strings.TrimSpace(replyAuthor) == "" {
		replyAuthor = "匿名用户"
	}
	if strings.TrimSpace(replyExcerpt) == "" {
		replyExcerpt = "（无文字摘要）"
	}
	if strings.HasPrefix(postURL, "/") {
		postURL = strings.TrimRight(cfg.AppBaseURL, "/") + postURL
	}

	content := fmt.Sprintf(`**匿名社区新回复**

> **帖子**：%s
> **来自**：%s
> **摘要**：%s

[点击查看完整内容](%s)`, postTitle, replyAuthor, replyExcerpt, postURL)

	msgReq := &schema.WeComAppMessageReq{
		ToUser:  lookupResp.UserID,
		MsgType: "markdown",
		AgentID: agentID,
	}
	msgReq.Markdown.Content = content

	msgResp := &schema.WeComSendMessageResp{}
	sendHTTPResp, err := s.httpClient.R().
		SetContext(ctx).
		SetQueryParam("access_token", token).
		SetBody(msgReq).
		SetResult(msgResp).
		Post("https://qyapi.weixin.qq.com/cgi-bin/message/send")
	if err != nil {
		return err
	}
	if sendHTTPResp.IsError() {
		return fmt.Errorf("wecom send message failed: status=%d body=%s", sendHTTPResp.StatusCode(), strings.TrimSpace(sendHTTPResp.String()))
	}
	if msgResp.ErrCode != 0 {
		return fmt.Errorf("wecom send message failed: errcode=%d errmsg=%s", msgResp.ErrCode, msgResp.ErrMsg)
	}
	return nil
}

func loadConfig() (*config, error) {
	cfg := &config{
		CorpID:           os.Getenv("WECOM_CORP_ID"),
		AgentID:          os.Getenv("WECOM_AGENT_ID"),
		AppSecret:        os.Getenv("WECOM_APP_SECRET"),
		AppBaseURL:       os.Getenv("APP_BASE_URL"),
		DefaultReturnTo:  os.Getenv("WECOM_DEFAULT_RETURN_TO"),
		VaultBaseURL:     os.Getenv("VAULT_BASE_URL"),
		VaultSharedToken: os.Getenv("VAULT_SHARED_TOKEN"),
		CallbackToken:    os.Getenv("WECOM_CALLBACK_TOKEN"),
		CallbackAESKey:   os.Getenv("WECOM_CALLBACK_AES_KEY"),
	}
	if cfg.DefaultReturnTo == "" {
		cfg.DefaultReturnTo = "/community"
	}
	if cfg.CorpID == "" || cfg.AppSecret == "" || cfg.AppBaseURL == "" || cfg.VaultBaseURL == "" || cfg.VaultSharedToken == "" {
		return nil, errors.BadRequest(reason.ReadConfigFailed)
	}
	return cfg, nil
}

func isLinkedCorpUserID(userID string) bool {
	if userID == "" {
		return false
	}
	parts := strings.SplitN(userID, "/", 2)
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

func joinDepartmentIntIDs(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.Itoa(id))
	}
	return strings.Join(parts, ",")
}

type anonymousUserCenter struct{}

func (anonymousUserCenter) Info() plugin.Info {
	return plugin.Info{
		Name:        plugin.MakeTranslator(""),
		SlugName:    providerSlug,
		Description: plugin.MakeTranslator(""),
		Author:      "internal",
		Version:     "0.1.0",
		Link:        "",
	}
}

func (anonymousUserCenter) Description() plugin.UserCenterDesc {
	return plugin.UserCenterDesc{
		Name:                      "WeCom Anonymous",
		DisplayName:               plugin.MakeTranslator(""),
		LoginRedirectURL:          "/answer/api/v1/wecom/auth/start?return_to=/community",
		SignUpRedirectURL:         "/answer/api/v1/wecom/auth/start?return_to=/community",
		EnabledOriginalUserSystem: true,
		MustAuthEmailEnabled:      false,
	}
}

func (anonymousUserCenter) ControlCenterItems() []plugin.ControlCenter { return nil }
func (anonymousUserCenter) LoginCallback(*plugin.GinContext) (*plugin.UserCenterBasicUserInfo, error) {
	return nil, nil
}
func (anonymousUserCenter) SignUpCallback(*plugin.GinContext) (*plugin.UserCenterBasicUserInfo, error) {
	return nil, nil
}
func (anonymousUserCenter) UserInfo(string) (*plugin.UserCenterBasicUserInfo, error) { return nil, nil }
func (anonymousUserCenter) UserStatus(string) plugin.UserStatus                      { return plugin.UserStatusAvailable }
func (anonymousUserCenter) UserList([]string) ([]*plugin.UserCenterBasicUserInfo, error) {
	return nil, nil
}
func (anonymousUserCenter) UserSettings(string) (*plugin.SettingInfo, error) {
	return &plugin.SettingInfo{}, nil
}
func (anonymousUserCenter) PersonalBranding(string) []*plugin.PersonalBranding { return nil }
func (anonymousUserCenter) AfterLogin(string, string)                          {}
