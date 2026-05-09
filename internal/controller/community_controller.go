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

package controller

import (
	"net/http"
	"strings"

	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/base/middleware"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/base/translator"
	"github.com/apache/answer/internal/base/validator"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/action"
	communityservice "github.com/apache/answer/internal/service/community"
	"github.com/apache/answer/internal/service/content"
	"github.com/apache/answer/internal/service/permission"
	"github.com/apache/answer/internal/service/rank"
	wecomservice "github.com/apache/answer/internal/service/wecom"
	"github.com/apache/answer/pkg/uid"
	"github.com/gin-gonic/gin"
	"github.com/segmentfault/pacman/errors"
)

type CommunityController struct {
	communityService *communityservice.Service
	wecomService     *wecomservice.Service
	questionService  *content.QuestionService
	rankService      *rank.RankService
	actionService    *action.CaptchaService
	rateLimit        *middleware.RateLimitMiddleware
}

func NewCommunityController(
	communityService *communityservice.Service,
	wecomService *wecomservice.Service,
	questionService *content.QuestionService,
	rankService *rank.RankService,
	actionService *action.CaptchaService,
	rateLimit *middleware.RateLimitMiddleware,
) *CommunityController {
	return &CommunityController{
		communityService: communityService,
		wecomService:     wecomService,
		questionService:  questionService,
		rankService:      rankService,
		actionService:    actionService,
		rateLimit:        rateLimit,
	}
}

func (cc *CommunityController) GetHome(ctx *gin.Context) {
	req := &schema.CommunityHomeReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	if req.IncludeAll && !middleware.GetUserIsAdminModerator(ctx) {
		handler.HandleResponse(ctx, errors.Forbidden(reason.ForbiddenError), nil)
		return
	}
	req.LoginUserID = middleware.GetLoginUserIDFromContext(ctx)
	resp, err := cc.communityService.GetHome(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}

func (cc *CommunityController) CreateQuestion(ctx *gin.Context) {
	req := &schema.CommunityCreateQuestionReq{}
	errFields := handler.BindAndCheckReturnErr(ctx, req)
	if ctx.IsAborted() {
		return
	}

	reject, rejectKey := cc.rateLimit.DuplicateRequestRejection(ctx, req)
	if reject {
		return
	}
	defer func() {
		if ctx.Writer.Status() != http.StatusOK {
			cc.rateLimit.DuplicateRequestClear(ctx, rejectKey)
		}
	}()

	addReq := &schema.QuestionAdd{
		Title:       req.Title,
		Content:     req.Content,
		Tags:        req.Tags,
		CaptchaID:   req.CaptchaID,
		CaptchaCode: req.CaptchaCode,
		ChannelType: entity.QuestionChannelQA,
	}
	linkURLLimitUser, isAdmin, checkErrFields, err := cc.prepareTopicCreate(ctx, addReq)
	errFields = append(errFields, checkErrFields...)
	if err != nil {
		if len(errFields) > 0 {
			handler.HandleResponse(ctx, errors.BadRequest(reason.RequestFormatError), errFields)
			return
		}
		handler.HandleResponse(ctx, err, nil)
		return
	}

	userID := middleware.GetLoginUserIDFromContext(ctx)
	resp, err := cc.communityService.CreateQuestion(ctx, req, userID, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	if err != nil {
		if fieldErrs, ok := resp.([]*validator.FormErrorField); ok {
			errFields = append(errFields, fieldErrs...)
		}
	}
	if len(errFields) > 0 {
		handler.HandleResponse(ctx, errors.BadRequest(reason.RequestFormatError), errFields)
		return
	}
	if err == nil && (!isAdmin || !linkURLLimitUser) {
		cc.actionService.ActionRecordAdd(ctx, entity.CaptchaActionQuestion, userID)
	}
	handler.HandleResponse(ctx, err, resp)
}

func (cc *CommunityController) CreateDiscussion(ctx *gin.Context) {
	req := &schema.CommunityCreateDiscussionReq{}
	errFields := handler.BindAndCheckReturnErr(ctx, req)
	if ctx.IsAborted() {
		return
	}

	reject, rejectKey := cc.rateLimit.DuplicateRequestRejection(ctx, req)
	if reject {
		return
	}
	defer func() {
		if ctx.Writer.Status() != http.StatusOK {
			cc.rateLimit.DuplicateRequestClear(ctx, rejectKey)
		}
	}()

	addReq := &schema.QuestionAdd{
		Title:       discussionTitle(req.Title),
		Content:     req.Content,
		Tags:        req.Tags,
		CaptchaID:   req.CaptchaID,
		CaptchaCode: req.CaptchaCode,
		ChannelType: entity.QuestionChannelDiscussion,
	}
	linkURLLimitUser, isAdmin, checkErrFields, err := cc.prepareTopicCreate(ctx, addReq)
	errFields = append(errFields, checkErrFields...)
	if err != nil {
		if len(errFields) > 0 {
			handler.HandleResponse(ctx, errors.BadRequest(reason.RequestFormatError), errFields)
			return
		}
		handler.HandleResponse(ctx, err, nil)
		return
	}

	userID := middleware.GetLoginUserIDFromContext(ctx)
	resp, err := cc.communityService.CreateDiscussion(ctx, req, userID, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	if err != nil {
		if fieldErrs, ok := resp.([]*validator.FormErrorField); ok {
			errFields = append(errFields, fieldErrs...)
		}
	}
	if len(errFields) > 0 {
		handler.HandleResponse(ctx, errors.BadRequest(reason.RequestFormatError), errFields)
		return
	}
	if err == nil && (!isAdmin || !linkURLLimitUser) {
		cc.actionService.ActionRecordAdd(ctx, entity.CaptchaActionQuestion, userID)
	}
	handler.HandleResponse(ctx, err, resp)
}

func (cc *CommunityController) GetQuestionDetail(ctx *gin.Context) {
	id := uid.DeShortID(ctx.Param("id"))
	resp, err := cc.communityService.GetQuestionDetail(ctx, id, middleware.GetLoginUserIDFromContext(ctx), entity.QuestionChannelQA)
	handler.HandleResponse(ctx, err, resp)
}

func (cc *CommunityController) GetDiscussionDetail(ctx *gin.Context) {
	id := uid.DeShortID(ctx.Param("id"))
	resp, err := cc.communityService.GetQuestionDetail(ctx, id, middleware.GetLoginUserIDFromContext(ctx), entity.QuestionChannelDiscussion)
	handler.HandleResponse(ctx, err, resp)
}

func (cc *CommunityController) CreateReply(ctx *gin.Context) {
	req := &schema.CommunityCreateReplyReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}

	reject, rejectKey := cc.rateLimit.DuplicateRequestRejection(ctx, req)
	if reject {
		return
	}
	defer func() {
		if ctx.Writer.Status() != http.StatusOK {
			cc.rateLimit.DuplicateRequestClear(ctx, rejectKey)
		}
	}()

	questionID := uid.DeShortID(ctx.Param("questionId"))
	userID := middleware.GetLoginUserIDFromContext(ctx)
	canList, err := cc.rankService.CheckOperationPermissions(ctx, userID, []string{
		permission.AnswerEdit,
		permission.AnswerDelete,
		permission.LinkUrlLimit,
	})
	if err != nil {
		handler.HandleResponse(ctx, err, nil)
		return
	}
	linkURLLimitUser := canList[2]
	isAdmin := middleware.GetUserIsAdminModerator(ctx)
	if !isAdmin || !linkURLLimitUser {
		captchaPass := cc.actionService.ActionRecordVerifyCaptcha(ctx, entity.CaptchaActionAnswer, userID, req.CaptchaID, req.CaptchaCode)
		if !captchaPass {
			handler.HandleResponse(ctx, errors.BadRequest(reason.CaptchaVerificationFailed), []*validator.FormErrorField{{
				ErrorField: "captcha_code",
				ErrorMsg:   translator.Tr(handler.GetLangByCtx(ctx), reason.CaptchaVerificationFailed),
			}})
			return
		}
	}
	can, err := cc.rankService.CheckOperationPermission(ctx, userID, permission.AnswerAdd, "")
	if err != nil {
		handler.HandleResponse(ctx, err, nil)
		return
	}
	if !can {
		handler.HandleResponse(ctx, errors.Forbidden(reason.RankFailToMeetTheCondition), nil)
		return
	}

	resp, err := cc.communityService.CreateReply(ctx, questionID, req, userID, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	if err == nil && (!isAdmin || !linkURLLimitUser) {
		cc.actionService.ActionRecordAdd(ctx, entity.CaptchaActionAnswer, userID)
	}
	handler.HandleResponse(ctx, err, resp)
}

func (cc *CommunityController) CreateComment(ctx *gin.Context) {
	req := &schema.CommunityCreateCommentReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}

	reject, rejectKey := cc.rateLimit.DuplicateRequestRejection(ctx, req)
	if reject {
		return
	}
	defer func() {
		if ctx.Writer.Status() != http.StatusOK {
			cc.rateLimit.DuplicateRequestClear(ctx, rejectKey)
		}
	}()

	answerID := ctx.Param("answerId")
	req.ReplyCommentID = decodeOptionalCommunityID(req.ReplyCommentID)
	userID := middleware.GetLoginUserIDFromContext(ctx)
	canList, err := cc.rankService.CheckOperationPermissions(ctx, userID, []string{
		permission.CommentAdd,
		permission.CommentEdit,
		permission.CommentDelete,
		permission.LinkUrlLimit,
	})
	if err != nil {
		handler.HandleResponse(ctx, err, nil)
		return
	}
	linkURLLimitUser := canList[3]
	isAdmin := middleware.GetUserIsAdminModerator(ctx)
	if !isAdmin || !linkURLLimitUser {
		captchaPass := cc.actionService.ActionRecordVerifyCaptcha(ctx, entity.CaptchaActionComment, userID, req.CaptchaID, req.CaptchaCode)
		if !captchaPass {
			handler.HandleResponse(ctx, errors.BadRequest(reason.CaptchaVerificationFailed), []*validator.FormErrorField{{
				ErrorField: "captcha_code",
				ErrorMsg:   translator.Tr(handler.GetLangByCtx(ctx), reason.CaptchaVerificationFailed),
			}})
			return
		}
	}
	if !canList[0] {
		handler.HandleResponse(ctx, errors.Forbidden(reason.RankFailToMeetTheCondition), nil)
		return
	}

	resp, err := cc.communityService.CreateComment(ctx, answerID, req, userID, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	if err == nil && (!isAdmin || !linkURLLimitUser) {
		cc.actionService.ActionRecordAdd(ctx, entity.CaptchaActionComment, userID)
	}
	handler.HandleResponse(ctx, err, resp)
}

func (cc *CommunityController) GetReplyComments(ctx *gin.Context) {
	req := &schema.CommunityReplyCommentPageReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}

	userID := middleware.GetLoginUserIDFromContext(ctx)
	canList, err := cc.rankService.CheckOperationPermissions(ctx, userID, []string{
		permission.CommentEdit,
		permission.CommentDelete,
	})
	if err != nil {
		handler.HandleResponse(ctx, err, nil)
		return
	}

	resp, err := cc.communityService.GetReplyComments(ctx, &schema.GetCommentWithPageReq{
		Page:      req.Page,
		PageSize:  req.PageSize,
		ObjectID:  ctx.Param("answerId"),
		CommentID: decodeOptionalCommunityID(req.CommentID),
		UserID:    userID,
		CanEdit:   canList[0],
		CanDelete: canList[1],
	})
	handler.HandleResponse(ctx, err, resp)
}

func (cc *CommunityController) AddReport(ctx *gin.Context) {
	req := &schema.AddReportReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}

	req.ObjectID = uid.DeShortID(req.ObjectID)
	userID := middleware.GetLoginUserIDFromContext(ctx)
	isAdmin := middleware.GetUserIsAdminModerator(ctx)
	if !isAdmin {
		captchaPass := cc.actionService.ActionRecordVerifyCaptcha(ctx, entity.CaptchaActionReport, userID, req.CaptchaID, req.CaptchaCode)
		if !captchaPass {
			handler.HandleResponse(ctx, errors.BadRequest(reason.CaptchaVerificationFailed), []*validator.FormErrorField{{
				ErrorField: "captcha_code",
				ErrorMsg:   translator.Tr(handler.GetLangByCtx(ctx), reason.CaptchaVerificationFailed),
			}})
			return
		}
	}

	can, err := cc.rankService.CheckOperationPermission(ctx, userID, permission.ReportAdd, "")
	if err != nil {
		handler.HandleResponse(ctx, err, nil)
		return
	}
	if !can {
		handler.HandleResponse(ctx, errors.Forbidden(reason.RankFailToMeetTheCondition), nil)
		return
	}

	err = cc.communityService.AddReport(ctx, req, userID)
	if err == nil && !isAdmin {
		cc.actionService.ActionRecordAdd(ctx, entity.CaptchaActionReport, userID)
	}
	handler.HandleResponse(ctx, err, nil)
}

func (cc *CommunityController) Moderate(ctx *gin.Context) {
	req := &schema.CommunityModerationActionReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.ObjectID = uid.DeShortID(req.ObjectID)
	err := cc.communityService.Moderate(ctx, req, middleware.GetLoginUserIDFromContext(ctx))
	handler.HandleResponse(ctx, err, nil)
}

func (cc *CommunityController) AuditReveal(ctx *gin.Context) {
	if !middleware.GetIsAdminFromContext(ctx) {
		handler.HandleResponse(ctx, errors.Forbidden(reason.ForbiddenError), nil)
		return
	}
	req := &schema.CommunityAuditRevealReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	resp, err := cc.wecomService.RevealIdentity(ctx, middleware.GetLoginUserIDFromContext(ctx), req)
	handler.HandleResponse(ctx, err, resp)
}

func (cc *CommunityController) PreviewBootstrap(ctx *gin.Context) {
	resp, err := cc.communityService.BootstrapPreview(ctx)
	handler.HandleResponse(ctx, err, resp)
}

func (cc *CommunityController) PreviewLogin(ctx *gin.Context) {
	resp, err := cc.communityService.PreviewLogin(ctx)
	handler.HandleResponse(ctx, err, resp)
}

func (cc *CommunityController) prepareTopicCreate(
	ctx *gin.Context,
	req *schema.QuestionAdd,
) (linkURLLimitUser, isAdmin bool, errFields []*validator.FormErrorField, err error) {
	userID := middleware.GetLoginUserIDFromContext(ctx)
	req.UserID = userID
	canList, requireRanks, err := cc.rankService.CheckOperationPermissionsForRanks(ctx, userID, []string{
		permission.QuestionAdd,
		permission.QuestionEdit,
		permission.QuestionDelete,
		permission.QuestionClose,
		permission.QuestionReopen,
		permission.TagUseReservedTag,
		permission.TagAdd,
		permission.LinkUrlLimit,
	})
	if err != nil {
		return false, false, nil, err
	}

	linkURLLimitUser = canList[7]
	isAdmin = middleware.GetUserIsAdminModerator(ctx)
	if !isAdmin || !linkURLLimitUser {
		captchaPass := cc.actionService.ActionRecordVerifyCaptcha(ctx, entity.CaptchaActionQuestion, userID, req.CaptchaID, req.CaptchaCode)
		if !captchaPass {
			errFields = append(errFields, &validator.FormErrorField{
				ErrorField: "captcha_code",
				ErrorMsg:   translator.Tr(handler.GetLangByCtx(ctx), reason.CaptchaVerificationFailed),
			})
			return linkURLLimitUser, isAdmin, errFields, errors.BadRequest(reason.CaptchaVerificationFailed)
		}
	}

	req.CanAdd = canList[0]
	req.CanEdit = canList[1]
	req.CanDelete = canList[2]
	req.CanClose = canList[3]
	req.CanReopen = canList[4]
	req.CanUseReservedTag = canList[5]
	req.CanAddTag = canList[6]
	if !req.CanAdd {
		return linkURLLimitUser, isAdmin, nil, errors.Forbidden(reason.RankFailToMeetTheCondition)
	}

	hasNewTag, err := cc.questionService.HasNewTag(ctx, req.Tags)
	if err != nil {
		return linkURLLimitUser, isAdmin, nil, err
	}
	if !req.CanAddTag && hasNewTag {
		msg := translator.TrWithData(handler.GetLangByCtx(ctx), reason.NoEnoughRankToOperate, &schema.PermissionTrTplData{
			Rank: requireRanks[6],
		})
		return linkURLLimitUser, isAdmin, nil, errors.Forbidden(reason.NoEnoughRankToOperate).WithMsg(msg)
	}

	if fieldErrs, err := cc.questionService.CheckAddQuestion(ctx, req); err != nil {
		if parsed, ok := fieldErrs.([]*validator.FormErrorField); ok {
			errFields = append(errFields, parsed...)
		}
		return linkURLLimitUser, isAdmin, errFields, err
	}

	return linkURLLimitUser, isAdmin, errFields, nil
}

func discussionTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "匿名讨论"
	}
	return title
}

func decodeOptionalCommunityID(id string) string {
	if strings.TrimSpace(id) == "" {
		return ""
	}
	return uid.DeShortID(id)
}
