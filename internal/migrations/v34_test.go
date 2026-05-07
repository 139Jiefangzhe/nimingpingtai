/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
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
	"testing"

	"github.com/apache/answer/internal/base/constant"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

func TestRemoveRankRequirementsForOrdinaryUserActions(t *testing.T) {
	t.Parallel()

	engine, err := xorm.NewEngine("sqlite", "file::memory:?cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, engine.Close())
	})

	require.NoError(t, engine.Sync(new(entity.Config), new(entity.SiteInfo)))

	configs := []*entity.Config{
		{ID: 1, Key: constant.RankQuestionAddKey, Value: "125"},
		{ID: 2, Key: constant.RankQuestionVoteDownKey, Value: "125"},
		{ID: 3, Key: constant.RankLinkUrlLimitKey, Value: "10"},
		{ID: 4, Key: constant.RankQuestionAuditKey, Value: "2000"},
	}
	_, err = engine.Insert(configs)
	require.NoError(t, err)

	privilegeContent, err := json.Marshal(&schema.UpdatePrivilegesConfigReq{
		Level: schema.PrivilegeLevelCustom,
		CustomPrivileges: []*constant.Privilege{
			{Key: constant.RankQuestionAddKey, Value: 125},
			{Key: constant.RankQuestionAuditKey, Value: 2000},
			{Key: constant.RankLinkUrlLimitKey, Value: 10},
		},
	})
	require.NoError(t, err)

	_, err = engine.Insert(&entity.SiteInfo{
		Type:    constant.SiteTypePrivileges,
		Content: string(privilegeContent),
		Status:  1,
	})
	require.NoError(t, err)

	require.NoError(t, removeRankRequirementsForOrdinaryUserActions(context.Background(), engine))

	for _, key := range []string{
		constant.RankQuestionAddKey,
		constant.RankQuestionVoteDownKey,
		constant.RankLinkUrlLimitKey,
	} {
		cfg := &entity.Config{Key: key}
		exist, err := engine.Get(cfg)
		require.NoError(t, err)
		require.True(t, exist)
		require.Equal(t, "0", cfg.Value)
	}

	auditCfg := &entity.Config{Key: constant.RankQuestionAuditKey}
	exist, err := engine.Get(auditCfg)
	require.NoError(t, err)
	require.True(t, exist)
	require.Equal(t, "2000", auditCfg.Value)

	siteInfo := &entity.SiteInfo{}
	exist, err = engine.Where("type = ?", constant.SiteTypePrivileges).Get(siteInfo)
	require.NoError(t, err)
	require.True(t, exist)

	updated := &schema.UpdatePrivilegesConfigReq{}
	require.NoError(t, json.Unmarshal([]byte(siteInfo.Content), updated))
	require.Equal(t, schema.PrivilegeLevelCustom, updated.Level)
	require.Len(t, updated.CustomPrivileges, 3)

	values := make(map[string]int, len(updated.CustomPrivileges))
	for _, privilege := range updated.CustomPrivileges {
		values[privilege.Key] = privilege.Value
	}

	require.Equal(t, 0, values[constant.RankQuestionAddKey])
	require.Equal(t, 0, values[constant.RankLinkUrlLimitKey])
	require.Equal(t, 2000, values[constant.RankQuestionAuditKey])
}
