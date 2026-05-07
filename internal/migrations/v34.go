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

package migrations

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	"xorm.io/xorm"
)

var ordinaryUserRankConfigKeys = []string{
	constant.RankQuestionAddKey,
	constant.RankAnswerAddKey,
	constant.RankCommentAddKey,
	constant.RankReportAddKey,
	constant.RankTagAddKey,
	constant.RankQuestionVoteUpKey,
	constant.RankQuestionVoteDownKey,
	constant.RankAnswerVoteUpKey,
	constant.RankAnswerVoteDownKey,
	constant.RankCommentVoteUpKey,
	constant.RankCommentVoteDownKey,
	constant.RankInviteSomeoneToAnswerKey,
	constant.RankLinkUrlLimitKey,
}

func removeRankRequirementsForOrdinaryUserActions(ctx context.Context, x *xorm.Engine) error {
	if err := zeroOrdinaryUserRankConfigValues(ctx, x); err != nil {
		return fmt.Errorf("zero ordinary rank config values failed: %w", err)
	}
	if err := zeroOrdinaryUserCustomPrivileges(ctx, x); err != nil {
		return fmt.Errorf("zero ordinary custom privileges failed: %w", err)
	}
	return nil
}

func zeroOrdinaryUserRankConfigValues(ctx context.Context, x *xorm.Engine) error {
	for _, key := range ordinaryUserRankConfigKeys {
		cfg := &entity.Config{Key: key}
		exist, err := x.Context(ctx).Get(cfg)
		if err != nil {
			return fmt.Errorf("get config %s failed: %w", key, err)
		}
		if !exist || cfg.Value == "0" {
			continue
		}

		if _, err = x.Context(ctx).ID(cfg.ID).Cols("value").Update(&entity.Config{Value: "0"}); err != nil {
			return fmt.Errorf("update config %s failed: %w", key, err)
		}
	}
	return nil
}

func zeroOrdinaryUserCustomPrivileges(ctx context.Context, x *xorm.Engine) error {
	siteInfo := &entity.SiteInfo{}
	exist, err := x.Context(ctx).Where("type = ?", constant.SiteTypePrivileges).Get(siteInfo)
	if err != nil {
		return fmt.Errorf("get privileges site info failed: %w", err)
	}
	if !exist || len(siteInfo.Content) == 0 {
		return nil
	}

	privilegeConfig := &schema.UpdatePrivilegesConfigReq{}
	if err := json.Unmarshal([]byte(siteInfo.Content), privilegeConfig); err != nil {
		return fmt.Errorf("unmarshal privileges site info failed: %w", err)
	}
	if len(privilegeConfig.CustomPrivileges) == 0 {
		return nil
	}

	changed := false
	for _, privilege := range privilegeConfig.CustomPrivileges {
		if isOrdinaryUserRankConfigKey(privilege.Key) && privilege.Value != 0 {
			privilege.Value = 0
			changed = true
		}
	}
	if !changed {
		return nil
	}

	content, err := json.Marshal(privilegeConfig)
	if err != nil {
		return fmt.Errorf("marshal privileges site info failed: %w", err)
	}

	verified := &schema.UpdatePrivilegesConfigReq{}
	if err := json.Unmarshal(content, verified); err != nil {
		return fmt.Errorf("verify privileges site info failed: %w", err)
	}

	if _, err = x.Context(ctx).
		Where("id = ?", siteInfo.ID).
		Cols("content").
		Update(&entity.SiteInfo{Content: string(content)}); err != nil {
		return fmt.Errorf("update privileges site info failed: %w", err)
	}
	return nil
}

func isOrdinaryUserRankConfigKey(key string) bool {
	for _, ordinaryKey := range ordinaryUserRankConfigKeys {
		if ordinaryKey == key {
			return true
		}
	}
	return false
}
