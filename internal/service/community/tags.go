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
	"fmt"
	"sync"

	"github.com/apache/answer/internal/base/reason"
	"github.com/apache/answer/internal/entity"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/pkg/converter"
	"github.com/apache/answer/pkg/uid"
	"github.com/segmentfault/pacman/errors"
)

var communityTagsMu sync.Mutex

// EnsureCommunityTags keeps the fixed community composer tags available before
// permission checks decide whether a post contains new tags.
func (s *Service) EnsureCommunityTags(ctx context.Context, userID string) ([]*schema.TagItem, error) {
	communityTagsMu.Lock()
	defer communityTagsMu.Unlock()

	items := make([]*schema.TagItem, 0, len(schema.AllowedQuestionTags))
	for _, name := range schema.AllowedQuestionTags {
		description := name + "相关匿名社区内容。"
		parsedText := converter.Markdown2HTML(description)
		tag := &entity.Tag{}
		exist, err := s.data.DB.Context(ctx).Where("slug_name = ?", name).Get(tag)
		if err != nil {
			return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
		}
		if !exist {
			tag = &entity.Tag{
				ID:           fmt.Sprintf("%d", uid.ID()),
				SlugName:     name,
				DisplayName:  name,
				OriginalText: description,
				ParsedText:   parsedText,
				Status:       entity.TagStatusAvailable,
				Recommend:    true,
				RevisionID:   "0",
				UserID:       userID,
			}
			if _, err = s.data.DB.Context(ctx).Insert(tag); err != nil {
				return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
			}
		} else {
			cols := make([]string, 0, 5)
			if tag.Status != entity.TagStatusAvailable {
				tag.Status = entity.TagStatusAvailable
				cols = append(cols, "status")
			}
			if !tag.Recommend {
				tag.Recommend = true
				cols = append(cols, "recommend")
			}
			if tag.DisplayName == "" {
				tag.DisplayName = name
				cols = append(cols, "display_name")
			}
			if tag.OriginalText == "" {
				tag.OriginalText = description
				tag.ParsedText = parsedText
				cols = append(cols, "original_text", "parsed_text")
			}
			if len(cols) > 0 {
				if _, err = s.data.DB.Context(ctx).ID(tag.ID).Cols(cols...).Update(tag); err != nil {
					return nil, errors.InternalServer(reason.DatabaseError).WithError(err).WithStack()
				}
			}
		}

		items = append(items, &schema.TagItem{
			SlugName:     name,
			DisplayName:  name,
			OriginalText: description,
			ParsedText:   parsedText,
		})
	}
	return items, nil
}
