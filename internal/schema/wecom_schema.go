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

package schema

type WeComAuthStartResp struct {
	AuthorizationURL string `json:"authorization_url"`
}

type WeComAuthCallbackResp struct {
	AccessToken   string `json:"access_token"`
	RedirectURL   string `json:"redirect_url"`
	AnonSubjectID string `json:"anon_subject_id"`
}

type WeComAppMessageReq struct {
	ToUser   string `json:"touser"`
	MsgType  string `json:"msgtype"`
	AgentID  int    `json:"agentid"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown"`
}

type WeComSendMessageResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type VaultResolveRequest struct {
	CorpID      string `json:"corp_id"`
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Avatar      string `json:"avatar"`
	Email       string `json:"email"`
	Mobile      string `json:"mobile"`
	Department  string `json:"department"`
	Position    string `json:"position"`
}

type VaultResolveResponse struct {
	AnonSubjectID string `json:"anon_subject_id"`
	DisplayName   string `json:"display_name"`
	Avatar        string `json:"avatar"`
	AvatarSeed    string `json:"avatar_seed"`
	Status        string `json:"status"`
}

type VaultStatusRequest struct {
	AnonSubjectID string `json:"anon_subject_id"`
}

type VaultStatusResponse struct {
	AnonSubjectID string `json:"anon_subject_id"`
	Status        string `json:"status"`
}

type VaultLookupResponse struct {
	CorpID        string `json:"corp_id"`
	UserID        string `json:"user_id"`
	AnonSubjectID string `json:"anon_subject_id"`
	Status        string `json:"status"`
}

type VaultUpdateStatusRequest struct {
	AnonSubjectID string `json:"anon_subject_id"`
	CorpID        string `json:"corp_id"`
	UserID        string `json:"user_id"`
	Status        string `json:"status"`
	Reason        string `json:"reason"`
	Metadata      string `json:"metadata"`
}

type VaultUpdateStatusResponse struct {
	AnonSubjectID string `json:"anon_subject_id"`
	Status        string `json:"status"`
}

type VaultRevealRequest struct {
	RequesterUserID string `json:"requester_user_id"`
	AnonSubjectID   string `json:"anon_subject_id"`
	Reason          string `json:"reason"`
}

type VaultRevealResponse struct {
	AnonSubjectID string `json:"anon_subject_id"`
	CorpID        string `json:"corp_id"`
	UserID        string `json:"user_id"`
	DisplayName   string `json:"display_name"`
	Email         string `json:"email"`
	Mobile        string `json:"mobile"`
	Status        string `json:"status"`
}

type VaultAuditLogRequest struct {
	RequesterUserID string `json:"requester_user_id"`
	AnonSubjectID   string `json:"anon_subject_id"`
	Action          string `json:"action"`
	Reason          string `json:"reason"`
	Metadata        string `json:"metadata"`
}
