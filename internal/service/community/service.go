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

package community

import (
	"context"
	"strings"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/base/data"
	"github.com/apache/answer/internal/base/pager"
	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	commentservice "github.com/apache/answer/internal/service/comment"
	"github.com/apache/answer/internal/service/content"
	questioncommon "github.com/apache/answer/internal/service/question_common"
	reportservice "github.com/apache/answer/internal/service/report"
	userexternallogin "github.com/apache/answer/internal/service/user_external_login"
	"github.com/apache/answer/pkg/htmltext"
	"github.com/apache/answer/pkg/uid"
	"github.com/segmentfault/pacman/errors"
)

type Service struct {
	data                   *data.Data
	questionService        *content.QuestionService
	answerService          *content.AnswerService
	commentService         *commentservice.CommentService
	reportService          *reportservice.ReportService
	questionCommon         *questioncommon.QuestionCommon
	userCenterLoginService *userexternallogin.UserCenterLoginService
}

func NewCommunityService(
	data *data.Data,
	questionService *content.QuestionService,
	answerService *content.AnswerService,
	commentService *commentservice.CommentService,
	reportService *reportservice.ReportService,
	questionCommon *questioncommon.QuestionCommon,
	userCenterLoginService *userexternallogin.UserCenterLoginService,
) *Service {
	return &Service{
		data:                   data,
		questionService:        questionService,
		answerService:          answerService,
		commentService:         commentService,
		reportService:          reportService,
		questionCommon:         questionCommon,
		userCenterLoginService: userCenterLoginService,
	}
}

func (s *Service) GetHome(ctx context.Context, req *schema.CommunityHomeReq) (*pager.PageModel, error) {
	page, pageSize := pager.ValPageAndPageSize(req.Page, req.PageSize)
	orderCond := req.OrderCond
	if orderCond == "" {
		orderCond = schema.QuestionOrderCondNewest
	}
	channel := req.Channel
	if channel == "" {
		channel = entity.QuestionChannelDiscussion
	}

	questions := make([]*entity.Question, 0)
	session := s.data.DB.Context(ctx).Table("question")
	status := []int{entity.QuestionStatusAvailable, entity.QuestionStatusClosed}
	if !req.IncludeAll {
		session.Where("question.show = ?", entity.QuestionShow)
	}
	if orderCond == schema.QuestionOrderCondUnanswered && !req.IncludeAll {
		status = []int{entity.QuestionStatusAvailable}
		session.And("question.answer_count = 0")
	}
	if req.IncludeAll {
		status = []int{entity.QuestionStatusAvailable, entity.QuestionStatusClosed, entity.QuestionStatusDeleted}
	}
	session.In("question.status", status)
	session.And("question.channel_type = ?", channel)

	switch orderCond {
	case schema.QuestionOrderCondActive:
		session.OrderBy("question.pin desc,question.post_update_time DESC, question.updated_at DESC")
	case schema.QuestionOrderCondHot:
		session.OrderBy("question.pin desc,question.hot_score DESC")
	case schema.QuestionOrderCondScore:
		session.OrderBy("question.pin desc,question.vote_count DESC, question.view_count DESC")
	case schema.QuestionOrderCondFrequent:
		session.OrderBy("question.pin DESC, question.linked_count DESC, question.updated_at DESC")
	default:
		session.OrderBy("question.pin desc,question.created_at DESC")
	}

	total, err := pager.Help(page, pageSize, &questions, &entity.Question{}, session)
	if err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	formatted, err := s.questionCommon.FormatQuestionsPage(ctx, questions, req.LoginUserID, orderCond)
	if err != nil {
		return nil, err
	}
	if err = s.decorateQuestionPages(ctx, formatted); err != nil {
		return nil, err
	}
	return pager.NewPageModel(total, formatted), nil
}

func (s *Service) CreateQuestion(ctx context.Context, req *schema.CommunityCreateQuestionReq, userID, ip, userAgent string) (any, error) {
	addReq := &schema.QuestionAdd{
		Title:       req.Title,
		Content:     req.Content,
		Tags:        req.Tags,
		CaptchaID:   req.CaptchaID,
		CaptchaCode: req.CaptchaCode,
		UserID:      userID,
		IP:          ip,
		UserAgent:   userAgent,
		ChannelType: entity.QuestionChannelQA,
	}
	if _, err := addReq.Check(); err != nil {
		return nil, err
	}
	return s.questionService.AddQuestion(ctx, addReq)
}

func (s *Service) CreateDiscussion(ctx context.Context, req *schema.CommunityCreateDiscussionReq, userID, ip, userAgent string) (any, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = makeDiscussionTitle(req.Content)
	}
	addReq := &schema.QuestionAdd{
		Title:       title,
		Content:     req.Content,
		Tags:        req.Tags,
		CaptchaID:   req.CaptchaID,
		CaptchaCode: req.CaptchaCode,
		UserID:      userID,
		IP:          ip,
		UserAgent:   userAgent,
		ChannelType: entity.QuestionChannelDiscussion,
	}
	if _, err := addReq.Check(); err != nil {
		return nil, err
	}
	return s.questionService.AddQuestion(ctx, addReq)
}

func (s *Service) GetQuestionDetail(ctx context.Context, id, loginUserID, channel string) (*schema.CommunityDetailResp, error) {
	question, err := s.questionService.GetQuestion(ctx, id, loginUserID, schema.QuestionPermission{})
	if err != nil {
		return nil, err
	}
	if question == nil || question.ChannelType != channel {
		return nil, errors.NotFound(reason.QuestionNotFound)
	}

	answerOrder := entity.AnswerSearchOrderByDefault
	if channel == entity.QuestionChannelDiscussion {
		answerOrder = entity.AnswerSearchOrderByTimeAsc
	}
	answerReq := &schema.AnswerListReq{
		QuestionID: id,
		Order:      answerOrder,
		Page:       1,
		PageSize:   20,
		UserID:     loginUserID,
	}
	replies, total, err := s.answerService.SearchList(ctx, answerReq)
	if err != nil {
		return nil, err
	}

	resp := &schema.CommunityDetailResp{
		Question:   question,
		Replies:    replies,
		ReplyCount: total,
	}
	if err = s.decorateDetail(ctx, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) CreateReply(ctx context.Context, questionID string, req *schema.CommunityCreateReplyReq, userID, ip, userAgent string) (*schema.AnswerInfo, error) {
	addReq := &schema.AnswerAddReq{
		QuestionID:  questionID,
		Content:     req.Content,
		CaptchaID:   req.CaptchaID,
		CaptchaCode: req.CaptchaCode,
		UserID:      userID,
		IP:          ip,
		UserAgent:   userAgent,
	}
	if _, err := addReq.Check(); err != nil {
		return nil, err
	}
	answerID, err := s.answerService.Insert(ctx, addReq)
	if err != nil {
		return nil, err
	}
	answer, _, _, err := s.answerService.Get(ctx, answerID, userID)
	if err != nil {
		return nil, err
	}
	if err = s.decorateReplies(ctx, []*schema.AnswerInfo{answer}); err != nil {
		return nil, err
	}
	return answer, nil
}

func (s *Service) CreateComment(ctx context.Context, answerID string, req *schema.CommunityCreateCommentReq, userID, ip, userAgent string) (*schema.GetCommentResp, error) {
	addReq := &schema.AddCommentReq{
		ObjectID:            answerID,
		ReplyCommentID:      req.ReplyCommentID,
		OriginalText:        req.OriginalText,
		UserID:              userID,
		CaptchaID:           req.CaptchaID,
		CaptchaCode:         req.CaptchaCode,
		IP:                  ip,
		UserAgent:           userAgent,
		MentionUsernameList: []string{},
	}
	if _, err := addReq.Check(); err != nil {
		return nil, err
	}
	comment, err := s.commentService.AddComment(ctx, addReq)
	if err != nil {
		return nil, err
	}
	if err = s.decorateComments(ctx, []*schema.GetCommentResp{comment}); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *Service) GetReplyComments(ctx context.Context, req *schema.GetCommentWithPageReq) (*pager.PageModel, error) {
	pageModel, err := s.commentService.GetCommentWithPage(ctx, req)
	if err != nil {
		return nil, err
	}
	items, ok := pageModel.List.([]*schema.GetCommentResp)
	if !ok {
		return pageModel, nil
	}
	if err = s.decorateComments(ctx, items); err != nil {
		return nil, err
	}
	return pager.NewPageModel(pageModel.Count, items), nil
}

func (s *Service) AddReport(ctx context.Context, req *schema.AddReportReq, userID string) error {
	req.UserID = userID
	return s.reportService.AddReport(ctx, req)
}

func (s *Service) Moderate(ctx context.Context, req *schema.CommunityModerationActionReq, actorUserID string) error {
	objectID := uid.DeShortID(req.ObjectID)
	session := s.data.DB.Context(ctx)
	var (
		actionRecord = &entity.ModerationAction{
			ActorUserID: actorUserID,
			ObjectType:  req.ObjectType,
			ObjectID:    objectID,
			Action:      req.Action,
			Reason:      req.Reason,
		}
		affected int64
		err      error
	)

	switch req.ObjectType {
	case constant.QuestionObjectType:
		update := &entity.Question{}
		cols := make([]string, 0, 3)
		switch req.Action {
		case "hide":
			update.Show = entity.QuestionHide
			update.ModerationState = entity.QuestionModerationStateBlocked
			cols = append(cols, "show", "moderation_state")
		case "unhide":
			update.Show = entity.QuestionShow
			update.ModerationState = entity.QuestionModerationStateNormal
			cols = append(cols, "show", "moderation_state")
		case "delete":
			update.Status = entity.QuestionStatusDeleted
			update.ModerationState = entity.QuestionModerationStateBlocked
			cols = append(cols, "status", "moderation_state")
		case "restore":
			update.Status = entity.QuestionStatusAvailable
			update.ModerationState = entity.QuestionModerationStateNormal
			cols = append(cols, "status", "moderation_state")
		}
		affected, err = session.ID(objectID).Cols(cols...).Update(update)
	case constant.AnswerObjectType:
		update := &entity.Answer{}
		switch req.Action {
		case "delete":
			update.Status = entity.AnswerStatusDeleted
		case "restore":
			update.Status = entity.AnswerStatusAvailable
		default:
			return errors.BadRequest(reason.RequestFormatError)
		}
		affected, err = session.ID(objectID).Cols("status").Update(update)
	case constant.CommentObjectType:
		update := &entity.Comment{}
		switch req.Action {
		case "delete":
			update.Status = entity.CommentStatusDeleted
		case "restore":
			update.Status = entity.CommentStatusAvailable
		default:
			return errors.BadRequest(reason.RequestFormatError)
		}
		affected, err = session.ID(objectID).Cols("status").Update(update)
	default:
		return errors.BadRequest(reason.RequestFormatError)
	}
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if affected == 0 {
		return errors.NotFound(reason.ObjectNotFound)
	}
	if _, err = session.Insert(actionRecord); err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func makeDiscussionTitle(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "匿名讨论"
	}
	title := htmltext.FetchExcerpt(content, "...", 32)
	if title == "" {
		return "匿名讨论"
	}
	if len(title) < 6 {
		return title + " 讨论"
	}
	return title
}
