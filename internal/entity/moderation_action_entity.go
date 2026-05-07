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

package entity

import "time"

// ModerationAction keeps an explicit moderation audit trail for anonymous community actions.
type ModerationAction struct {
	ID          int64     `xorm:"not null pk autoincr BIGINT(20) id"`
	CreatedAt   time.Time `xorm:"created TIMESTAMP created_at"`
	ActorUserID string    `xorm:"not null default 0 BIGINT(20) INDEX actor_user_id"`
	ObjectType  string    `xorm:"not null default '' VARCHAR(20) INDEX object_type"`
	ObjectID    string    `xorm:"not null default 0 BIGINT(20) INDEX object_id"`
	Action      string    `xorm:"not null default '' VARCHAR(32) action"`
	Reason      string    `xorm:"TEXT reason"`
	Metadata    string    `xorm:"TEXT metadata"`
}

func (ModerationAction) TableName() string {
	return "moderation_action"
}
