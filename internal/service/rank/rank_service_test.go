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

package rank

import (
	"testing"

	"github.com/apache/answer/internal/service/permission"
	"github.com/stretchr/testify/require"
)

func TestIsRankBypassedLoggedInAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action string
		want   bool
	}{
		{name: "question add", action: permission.QuestionAdd, want: true},
		{name: "answer add", action: permission.AnswerAdd, want: true},
		{name: "comment add", action: permission.CommentAdd, want: true},
		{name: "report add", action: permission.ReportAdd, want: true},
		{name: "tag add", action: permission.TagAdd, want: true},
		{name: "question vote up", action: permission.QuestionVoteUp, want: true},
		{name: "question vote down", action: permission.QuestionVoteDown, want: true},
		{name: "answer vote up", action: permission.AnswerVoteUp, want: true},
		{name: "answer vote down", action: permission.AnswerVoteDown, want: true},
		{name: "comment vote up", action: permission.CommentVoteUp, want: true},
		{name: "comment vote down", action: permission.CommentVoteDown, want: true},
		{name: "invite to answer", action: permission.AnswerInviteSomeoneToAnswer, want: true},
		{name: "link url limit", action: permission.LinkUrlLimit, want: true},
		{name: "question edit", action: permission.QuestionEdit, want: false},
		{name: "question close", action: permission.QuestionClose, want: false},
		{name: "answer edit without review", action: permission.AnswerEditWithoutReview, want: false},
		{name: "tag audit", action: permission.TagAudit, want: false},
		{name: "tag use reserved tag", action: permission.TagUseReservedTag, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, isRankBypassedLoggedInAction(tt.action))
		})
	}
}
