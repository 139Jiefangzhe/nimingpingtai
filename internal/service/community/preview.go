/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance
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
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/pkg/converter"
	"github.com/apache/answer/pkg/uid"
	"github.com/apache/answer/plugin"
	"github.com/segmentfault/pacman/errors"
)

const (
	previewModeLocal         = "local"
	previewProviderSlug      = "community-preview"
	previewSeedConfigKey     = "community.preview.seed.version"
	previewSeedVersion       = "1"
	previewPrimaryExternalID = "preview-anon-primary"
	previewSecondExternalID  = "preview-anon-secondary"
	previewSeedIP            = "127.0.0.1"
	previewSeedUA            = "community-preview-seed"
)

var previewSeedMu sync.Mutex

type previewIdentity struct {
	ExternalID string
	Username   string
	Display    string
	AvatarSeed string
}

func previewModeEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("COMMUNITY_PREVIEW_MODE")), previewModeLocal)
}

func previewIdentities() []previewIdentity {
	return []previewIdentity{
		{
			ExternalID: previewPrimaryExternalID,
			Username:   "previewanonprimary",
			Display:    "匿名用户A1F3",
			AvatarSeed: "A1F3D2C7",
		},
		{
			ExternalID: previewSecondExternalID,
			Username:   "previewanonsecondary",
			Display:    "匿名用户B7K2",
			AvatarSeed: "B7K2M9Q4",
		},
	}
}

func (s *Service) BootstrapPreview(ctx context.Context) (*schema.CommunityPreviewBootstrapResp, error) {
	if !previewModeEnabled() {
		return nil, errors.Forbidden(reason.ForbiddenError)
	}
	seeded, err := s.ensurePreviewSeedData(ctx)
	if err != nil {
		return nil, err
	}
	return &schema.CommunityPreviewBootstrapResp{
		Enabled: true,
		Mode:    previewModeLocal,
		Seeded:  seeded,
	}, nil
}

func (s *Service) PreviewLogin(ctx context.Context) (*schema.CommunityPreviewLoginResp, error) {
	if !previewModeEnabled() {
		return nil, errors.Forbidden(reason.ForbiddenError)
	}
	if _, err := s.ensurePreviewSeedData(ctx); err != nil {
		return nil, err
	}

	profile, accessToken, _, err := s.ensurePreviewUser(ctx, previewIdentities()[0])
	if err != nil {
		return nil, err
	}
	return &schema.CommunityPreviewLoginResp{
		AccessToken:   accessToken,
		RedirectURL:   "/users/auth-landing?access_token=" + url.QueryEscape(accessToken),
		AnonSubjectID: profile.AnonSubjectID,
		DisplayName:   profile.DisplayName,
		AvatarSeed:    profile.AvatarSeed,
	}, nil
}

func (s *Service) ensurePreviewSeedData(ctx context.Context) (bool, error) {
	previewSeedMu.Lock()
	defer previewSeedMu.Unlock()

	cfg := &entity.Config{}
	exist, err := s.data.DB.Context(ctx).Where("key = ?", previewSeedConfigKey).Get(cfg)
	if err != nil {
		return false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if exist && cfg.Value == previewSeedVersion {
		return true, nil
	}

	_, _, primaryUserID, err := s.ensurePreviewUser(ctx, previewIdentities()[0])
	if err != nil {
		return false, err
	}
	_, _, secondaryUserID, err := s.ensurePreviewUser(ctx, previewIdentities()[1])
	if err != nil {
		return false, err
	}

	tags, err := s.ensurePreviewTags(ctx, primaryUserID)
	if err != nil {
		return false, err
	}

	discussionIDs, err := s.seedPreviewDiscussions(ctx, primaryUserID, secondaryUserID, tags)
	if err != nil {
		return false, err
	}
	if _, err = s.seedPreviewQuestions(ctx, primaryUserID, secondaryUserID, tags); err != nil {
		return false, err
	}

	if len(discussionIDs) > 0 {
		_ = s.reportService.AddReport(ctx, &schema.AddReportReq{
			ObjectID:   discussionIDs[0],
			ReportType: 57,
			Content:    "演示环境自动生成的举报记录",
			UserID:     secondaryUserID,
		})
	}
	if len(discussionIDs) > 3 {
		if _, err = s.data.DB.Context(ctx).ID(discussionIDs[3]).Cols("show", "moderation_state").Update(&entity.Question{
			Show:            entity.QuestionHide,
			ModerationState: entity.QuestionModerationStateBlocked,
		}); err != nil {
			return false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
	}

	cfg.Key = previewSeedConfigKey
	cfg.Value = previewSeedVersion
	if exist {
		_, err = s.data.DB.Context(ctx).Where("key = ?", previewSeedConfigKey).Cols("value").Update(cfg)
	} else {
		_, err = s.data.DB.Context(ctx).Insert(cfg)
	}
	if err != nil {
		return false, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return true, nil
}

func (s *Service) ensurePreviewTags(ctx context.Context, userID string) ([]*schema.TagItem, error) {
	defs := []struct {
		slug string
		name string
		desc string
	}{
		{slug: "culture", name: "Culture", desc: "用于讨论团队文化和协作氛围。"},
		{slug: "workflow", name: "Workflow", desc: "用于讨论内部流程、制度和协作方式。"},
		{slug: "product", name: "Product", desc: "用于讨论产品体验、需求和FAQ沉淀。"},
	}

	items := make([]*schema.TagItem, 0, len(defs))
	for _, def := range defs {
		tag := &entity.Tag{}
		exist, err := s.data.DB.Context(ctx).Where("slug_name = ?", def.slug).Get(tag)
		if err != nil {
			return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
		if !exist {
			tag = &entity.Tag{
				ID:           fmt.Sprintf("%d", uid.ID()),
				SlugName:     def.slug,
				DisplayName:  def.name,
				OriginalText: def.desc,
				ParsedText:   converter.Markdown2HTML(def.desc),
				Status:       entity.TagStatusAvailable,
				Recommend:    true,
				RevisionID:   "0",
				UserID:       userID,
			}
			if _, err = s.data.DB.Context(ctx).Insert(tag); err != nil {
				return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
			}
		} else if !tag.Recommend {
			tag.Recommend = true
			if _, err = s.data.DB.Context(ctx).ID(tag.ID).Cols("recommend").Update(tag); err != nil {
				return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
			}
		}
		items = append(items, &schema.TagItem{
			SlugName:     def.slug,
			DisplayName:  def.name,
			OriginalText: def.desc,
		})
	}
	return items, nil
}

func (s *Service) seedPreviewDiscussions(ctx context.Context, primaryUserID, secondaryUserID string, tags []*schema.TagItem) ([]string, error) {
	posts := []struct {
		title   string
		content string
		userID  string
	}{
		{
			title:   "有没有比周会更轻量的同步方式？",
			content: "我们团队现在每周固定同步，但大家普遍觉得信息密度不高。匿名区里想听听真实建议：是改成异步周报、站会，还是保留周会但压缩到 15 分钟？",
			userID:  primaryUserID,
		},
		{
			title:   "匿名区上线后，最担心什么？",
			content: "如果匿名论坛正式上线，你最担心的是滥用、信息泄露，还是运营团队会过度介入？欢迎直接说问题，不用修饰。",
			userID:  secondaryUserID,
		},
		{
			title:   "大家最想先看到哪些社区规则？",
			content: "如果只保留三条社区规则，你觉得应该是什么？我个人偏向：不做人身攻击、不泄露敏感信息、举报必回执。",
			userID:  primaryUserID,
		},
		{
			title:   "这是一条被隐藏的讨论示例",
			content: "这条内容会在种子数据里被标记为隐藏，用来预览管理台里的审核操作和状态样式。",
			userID:  secondaryUserID,
		},
	}

	ids := make([]string, 0, len(posts))
	for index, post := range posts {
		resp, err := s.CreateDiscussion(ctx, &schema.CommunityCreateDiscussionReq{
			Title:   post.title,
			Content: post.content,
			Tags:    previewDiscussionTags(tags, index),
		}, post.userID, previewSeedIP, previewSeedUA)
		if err != nil {
			return nil, err
		}
		questionInfo, ok := resp.(*schema.QuestionInfoResp)
		if !ok || questionInfo == nil {
			return nil, errors.InternalServer(reason.UnknownError)
		}
		if err = s.forceQuestionAvailable(ctx, questionInfo.ID); err != nil {
			return nil, err
		}

		replyText := previewDiscussionReply(index)
		replyID, err := s.answerService.Insert(ctx, &schema.AnswerAddReq{
			QuestionID: questionInfo.ID,
			Content:    replyText,
			HTML:       converter.Markdown2HTML(replyText),
			UserID:     otherPreviewUser(post.userID, primaryUserID, secondaryUserID),
			IP:         previewSeedIP,
			UserAgent:  previewSeedUA,
		})
		if err != nil {
			return nil, err
		}
		if err = s.forceAnswerAvailable(ctx, replyID); err != nil {
			return nil, err
		}

		comment, err := s.commentService.AddComment(ctx, &schema.AddCommentReq{
			ObjectID:            replyID,
			OriginalText:        "这条评论也是演示数据，方便你直接预览楼中评论样式。",
			ParsedText:          converter.Markdown2HTML("这条评论也是演示数据，方便你直接预览楼中评论样式。"),
			UserID:              post.userID,
			IP:                  previewSeedIP,
			UserAgent:           previewSeedUA,
			MentionUsernameList: []string{},
		})
		if err == nil && comment != nil {
			if forceErr := s.forceCommentAvailable(ctx, comment.CommentID); forceErr != nil {
				return nil, forceErr
			}
		}
		ids = append(ids, questionInfo.ID)
	}
	return ids, nil
}

func (s *Service) seedPreviewQuestions(ctx context.Context, primaryUserID, secondaryUserID string, tags []*schema.TagItem) ([]string, error) {
	posts := []struct {
		title   string
		content string
		userID  string
		tags    []*schema.TagItem
	}{
		{
			title:   "新人入职第一周最需要补的知识是什么？",
			content: "如果只能给新人一个清单，你会优先放什么内容？我倾向于把业务背景、常见流程和内部术语表放在最前面。",
			userID:  primaryUserID,
			tags:    []*schema.TagItem{tags[0], tags[1]},
		},
		{
			title:   "如果要做内部 FAQ，标签怎么设计更合理？",
			content: "我们准备把匿名区里高质量问答沉淀成 FAQ。标签是按业务域分，还是按角色分，比如销售、运营、研发？",
			userID:  secondaryUserID,
			tags:    []*schema.TagItem{tags[1], tags[2]},
		},
		{
			title:   "匿名区的举报规则应该怎么定？",
			content: "举报规则既要保护表达，也要控制滥用。大家觉得处理时效、反馈方式、处罚等级，哪个最该先明确？",
			userID:  primaryUserID,
			tags:    []*schema.TagItem{tags[0], tags[2]},
		},
	}

	ids := make([]string, 0, len(posts))
	for _, post := range posts {
		resp, err := s.CreateQuestion(ctx, &schema.CommunityCreateQuestionReq{
			Title:   post.title,
			Content: post.content,
			Tags:    post.tags,
		}, post.userID, previewSeedIP, previewSeedUA)
		if err != nil {
			return nil, err
		}
		questionInfo, ok := resp.(*schema.QuestionInfoResp)
		if !ok || questionInfo == nil {
			return nil, errors.InternalServer(reason.UnknownError)
		}
		if err = s.forceQuestionAvailable(ctx, questionInfo.ID); err != nil {
			return nil, err
		}

		replyText := "这条回答由另一位演示匿名用户补充，用来展示问答频道的回复结构。"
		replyID, err := s.answerService.Insert(ctx, &schema.AnswerAddReq{
			QuestionID: questionInfo.ID,
			Content:    replyText,
			HTML:       converter.Markdown2HTML(replyText),
			UserID:     otherPreviewUser(post.userID, primaryUserID, secondaryUserID),
			IP:         previewSeedIP,
			UserAgent:  previewSeedUA,
		})
		if err != nil {
			return nil, err
		}
		if err = s.forceAnswerAvailable(ctx, replyID); err != nil {
			return nil, err
		}
		ids = append(ids, questionInfo.ID)
	}
	return ids, nil
}

func previewDiscussionTags(tags []*schema.TagItem, index int) []*schema.TagItem {
	if len(tags) < 3 {
		return tags
	}
	switch index {
	case 0:
		return []*schema.TagItem{tags[1]}
	case 1:
		return []*schema.TagItem{tags[0]}
	default:
		return []*schema.TagItem{tags[0], tags[2]}
	}
}

func previewDiscussionReply(index int) string {
	replies := []string{
		"我支持先试两周异步周报，再保留双周一次深度同步，节奏会更舒服。",
		"我最担心的是大家不信任匿名承诺，所以规则和审计边界要写清楚。",
		"最少三条规则完全够用，重点是执行一致，不要时紧时松。",
		"这条回复只是为了让隐藏帖也有完整的详情结构。",
	}
	return replies[index]
}

func otherPreviewUser(currentUserID, primaryUserID, secondaryUserID string) string {
	if currentUserID == primaryUserID {
		return secondaryUserID
	}
	return primaryUserID
}

func (s *Service) ensurePreviewUser(ctx context.Context, identity previewIdentity) (*entity.AnonymousProfile, string, string, error) {
	resp, err := s.userCenterLoginService.ExternalLogin(ctx, previewUserCenter{}, &plugin.UserCenterBasicUserInfo{
		ExternalID:  identity.ExternalID,
		Username:    identity.Username,
		DisplayName: identity.Display,
	})
	if err != nil {
		return nil, "", "", err
	}
	if resp == nil || resp.ErrMsg != "" {
		return nil, "", "", errors.BadRequest(reason.UserAccessDenied)
	}

	link := &entity.UserExternalLogin{}
	exist, err := s.data.DB.Context(ctx).
		Where("provider = ?", previewProviderSlug).
		And("external_id = ?", identity.ExternalID).
		Get(link)
	if err != nil {
		return nil, "", "", errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if !exist {
		return nil, "", "", errors.InternalServer(reason.DatabaseError)
	}

	profile := &entity.AnonymousProfile{UserID: link.UserID}
	has, err := s.data.DB.Context(ctx).Get(profile)
	if err != nil {
		return nil, "", "", errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	if has {
		profile.AnonSubjectID = identity.ExternalID
		profile.DisplayName = identity.Display
		profile.AvatarSeed = identity.AvatarSeed
		profile.Status = entity.AnonymousProfileStatusActive
		if _, err = s.data.DB.Context(ctx).ID(profile.UserID).Cols("anon_subject_id", "display_name", "avatar_seed", "status").Update(profile); err != nil {
			return nil, "", "", errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
	} else {
		profile.AnonSubjectID = identity.ExternalID
		profile.DisplayName = identity.Display
		profile.AvatarSeed = identity.AvatarSeed
		profile.Status = entity.AnonymousProfileStatusActive
		if _, err = s.data.DB.Context(ctx).Insert(profile); err != nil {
			return nil, "", "", errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
	}
	return profile, resp.AccessToken, link.UserID, nil
}

func (s *Service) forceQuestionAvailable(ctx context.Context, questionID string) error {
	_, err := s.data.DB.Context(ctx).ID(questionID).Cols("status", "show", "moderation_state").Update(&entity.Question{
		Status:          entity.QuestionStatusAvailable,
		Show:            entity.QuestionShow,
		ModerationState: entity.QuestionModerationStateNormal,
	})
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (s *Service) forceAnswerAvailable(ctx context.Context, answerID string) error {
	_, err := s.data.DB.Context(ctx).ID(answerID).Cols("status").Update(&entity.Answer{
		Status: entity.AnswerStatusAvailable,
	})
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (s *Service) forceCommentAvailable(ctx context.Context, commentID string) error {
	_, err := s.data.DB.Context(ctx).ID(commentID).Cols("status").Update(&entity.Comment{
		Status: entity.CommentStatusAvailable,
	})
	if err != nil {
		return errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	return nil
}

func (s *Service) decorateQuestionPages(ctx context.Context, items []*schema.QuestionPageResp) error {
	userIDs := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil && item.Operator != nil && item.Operator.ID != "" {
			userIDs = append(userIDs, item.Operator.ID)
		}
	}
	profiles, err := s.anonymousProfilesByUserID(ctx, userIDs)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item == nil || item.Operator == nil {
			continue
		}
		if profile, ok := profiles[item.Operator.ID]; ok {
			item.Operator.Anonymous = toAnonymousUserInfo(profile)
		}
	}
	return nil
}

func (s *Service) decorateDetail(ctx context.Context, detail *schema.CommunityDetailResp) error {
	if detail == nil {
		return nil
	}
	userIDs := make([]string, 0, len(detail.Replies)*2+3)
	if detail.Question != nil {
		userIDs = appendUserInfoIDs(userIDs, detail.Question.UserInfo, detail.Question.UpdateUserInfo, detail.Question.LastAnsweredUserInfo)
	}
	for _, reply := range detail.Replies {
		userIDs = appendUserInfoIDs(userIDs, reply.UserInfo, reply.UpdateUserInfo)
	}
	profiles, err := s.anonymousProfilesByUserID(ctx, userIDs)
	if err != nil {
		return err
	}
	if detail.Question != nil {
		applyAnonymousToUserInfo(detail.Question.UserInfo, profiles)
		applyAnonymousToUserInfo(detail.Question.UpdateUserInfo, profiles)
		applyAnonymousToUserInfo(detail.Question.LastAnsweredUserInfo, profiles)
	}
	for _, reply := range detail.Replies {
		applyAnonymousToUserInfo(reply.UserInfo, profiles)
		applyAnonymousToUserInfo(reply.UpdateUserInfo, profiles)
	}
	return nil
}

func (s *Service) decorateReplies(ctx context.Context, replies []*schema.AnswerInfo) error {
	userIDs := make([]string, 0, len(replies)*2)
	for _, reply := range replies {
		userIDs = appendUserInfoIDs(userIDs, reply.UserInfo, reply.UpdateUserInfo)
	}
	profiles, err := s.anonymousProfilesByUserID(ctx, userIDs)
	if err != nil {
		return err
	}
	for _, reply := range replies {
		applyAnonymousToUserInfo(reply.UserInfo, profiles)
		applyAnonymousToUserInfo(reply.UpdateUserInfo, profiles)
	}
	return nil
}

func (s *Service) decorateComments(ctx context.Context, comments []*schema.GetCommentResp) error {
	userIDs := make([]string, 0, len(comments)*2)
	for _, comment := range comments {
		if comment == nil {
			continue
		}
		if comment.UserID != "" {
			userIDs = append(userIDs, comment.UserID)
		}
		if comment.ReplyUserID != "" {
			userIDs = append(userIDs, comment.ReplyUserID)
		}
	}
	profiles, err := s.anonymousProfilesByUserID(ctx, userIDs)
	if err != nil {
		return err
	}
	for _, comment := range comments {
		if comment == nil {
			continue
		}
		if profile, ok := profiles[comment.UserID]; ok {
			comment.UserAnonymous = toAnonymousUserInfo(profile)
			comment.UserDisplayName = profile.DisplayName
		}
		if profile, ok := profiles[comment.ReplyUserID]; ok {
			comment.ReplyUserAnonymous = toAnonymousUserInfo(profile)
			comment.ReplyUserDisplayName = profile.DisplayName
		}
	}
	return nil
}

func (s *Service) anonymousProfilesByUserID(ctx context.Context, userIDs []string) (map[string]*entity.AnonymousProfile, error) {
	userIDs = filterEmptyStrings(userIDs)
	if len(userIDs) == 0 {
		return map[string]*entity.AnonymousProfile{}, nil
	}
	profiles := make([]*entity.AnonymousProfile, 0, len(userIDs))
	if err := s.data.DB.Context(ctx).In("user_id", userIDs).Find(&profiles); err != nil {
		return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
	}
	result := make(map[string]*entity.AnonymousProfile, len(profiles))
	for _, profile := range profiles {
		result[profile.UserID] = profile
	}
	return result, nil
}

func applyAnonymousToUserInfo(userInfo *schema.UserBasicInfo, profiles map[string]*entity.AnonymousProfile) {
	if userInfo == nil {
		return
	}
	if profile, ok := profiles[userInfo.ID]; ok {
		userInfo.Anonymous = toAnonymousUserInfo(profile)
	}
}

func toAnonymousUserInfo(profile *entity.AnonymousProfile) *schema.AnonymousUserInfo {
	if profile == nil {
		return nil
	}
	return &schema.AnonymousUserInfo{
		Enabled:       true,
		AnonSubjectID: profile.AnonSubjectID,
		DisplayName:   profile.DisplayName,
		AvatarSeed:    profile.AvatarSeed,
	}
}

func appendUserInfoIDs(ids []string, infos ...*schema.UserBasicInfo) []string {
	for _, info := range infos {
		if info != nil && info.ID != "" {
			ids = append(ids, info.ID)
		}
	}
	return ids
}

func filterEmptyStrings(items []string) []string {
	if len(items) == 0 {
		return items
	}
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

type previewUserCenter struct{}

func (previewUserCenter) Info() plugin.Info {
	return plugin.Info{
		Name:        plugin.MakeTranslator(""),
		SlugName:    previewProviderSlug,
		Description: plugin.MakeTranslator(""),
		Author:      "internal",
		Version:     "0.1.0",
		Link:        "",
	}
}

func (previewUserCenter) Description() plugin.UserCenterDesc {
	return plugin.UserCenterDesc{
		Name:                      "Community Preview",
		DisplayName:               plugin.MakeTranslator(""),
		LoginRedirectURL:          "/answer/api/v1/community/preview/login",
		SignUpRedirectURL:         "/answer/api/v1/community/preview/login",
		EnabledOriginalUserSystem: true,
		MustAuthEmailEnabled:      false,
	}
}

func (previewUserCenter) ControlCenterItems() []plugin.ControlCenter { return nil }
func (previewUserCenter) LoginCallback(*plugin.GinContext) (*plugin.UserCenterBasicUserInfo, error) {
	return nil, nil
}
func (previewUserCenter) SignUpCallback(*plugin.GinContext) (*plugin.UserCenterBasicUserInfo, error) {
	return nil, nil
}
func (previewUserCenter) UserInfo(string) (*plugin.UserCenterBasicUserInfo, error) { return nil, nil }
func (previewUserCenter) UserStatus(string) plugin.UserStatus                      { return plugin.UserStatusAvailable }
func (previewUserCenter) UserList([]string) ([]*plugin.UserCenterBasicUserInfo, error) {
	return nil, nil
}
func (previewUserCenter) UserSettings(string) (*plugin.SettingInfo, error) {
	return &plugin.SettingInfo{}, nil
}
func (previewUserCenter) PersonalBranding(string) []*plugin.PersonalBranding { return nil }
func (previewUserCenter) AfterLogin(string, string)                          {}
