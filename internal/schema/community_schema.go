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

type CommunityHomeReq struct {
	Page       int    `validate:"omitempty,min=1" form:"page"`
	PageSize   int    `validate:"omitempty,min=1" form:"page_size"`
	OrderCond  string `validate:"omitempty,oneof=newest active hot score unanswered frequent" form:"order"`
	Channel    string `validate:"omitempty,oneof=qa discussion" form:"channel"`
	IncludeAll bool   `form:"include_all"`

	LoginUserID string `json:"-"`
}

type CommunityCreateQuestionReq struct {
	Title   string     `validate:"required,notblank,gte=6,lte=150" json:"title"`
	Content string     `validate:"required,notblank,gte=6,lte=65535" json:"content"`
	Tags    []*TagItem `validate:"dive" json:"tags"`

	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

type CommunityCreateDiscussionReq struct {
	Title   string     `validate:"omitempty,lte=150" json:"title"`
	Content string     `validate:"required,notblank,gte=6,lte=65535" json:"content"`
	Tags    []*TagItem `validate:"omitempty,dive" json:"tags"`

	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

type CommunityCreateReplyReq struct {
	Content string `validate:"required,notblank,gte=6,lte=65535" json:"content"`

	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

type CommunityCreateCommentReq struct {
	OriginalText   string `validate:"required,notblank,gte=2,lte=600" json:"original_text"`
	ReplyCommentID string `validate:"omitempty" json:"reply_comment_id"`

	CaptchaID   string `json:"captcha_id"`
	CaptchaCode string `json:"captcha_code"`
}

type CommunityReplyCommentPageReq struct {
	Page      int    `validate:"omitempty,min=1" form:"page"`
	PageSize  int    `validate:"omitempty,min=1" form:"page_size"`
	CommentID string `validate:"omitempty" form:"comment_id"`
}

type CommunityDetailResp struct {
	Question   *QuestionInfoResp `json:"question"`
	Replies    []*AnswerInfo     `json:"replies"`
	ReplyCount int64             `json:"reply_count"`
}

type CommunityModerationActionReq struct {
	ObjectType string `validate:"required,oneof=question answer comment" json:"object_type"`
	ObjectID   string `validate:"required" json:"object_id"`
	Action     string `validate:"required,oneof=hide unhide delete restore" json:"action"`
	Reason     string `validate:"omitempty,lte=500" json:"reason"`
}

type CommunityAuditRevealReq struct {
	AnonSubjectID string `validate:"required" json:"anon_subject_id"`
	Reason        string `validate:"required,notblank,lte=500" json:"reason"`
}

type CommunityAuditRevealResp struct {
	AnonSubjectID string `json:"anon_subject_id"`
	CorpID        string `json:"corp_id"`
	UserID        string `json:"user_id"`
	DisplayName   string `json:"display_name"`
	Email         string `json:"email"`
	Mobile        string `json:"mobile"`
	Status        string `json:"status"`
}

type CommunityPreviewBootstrapResp struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"`
	Seeded  bool   `json:"seeded"`
}

type CommunityPreviewLoginResp struct {
	AccessToken   string `json:"access_token"`
	RedirectURL   string `json:"redirect_url"`
	AnonSubjectID string `json:"anon_subject_id"`
	DisplayName   string `json:"display_name"`
	AvatarSeed    string `json:"avatar_seed"`
}
