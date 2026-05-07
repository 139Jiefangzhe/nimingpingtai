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

const (
	AnonymousProfileStatusActive   = "active"
	AnonymousProfileStatusDisabled = "disabled"
)

// AnonymousProfile stores the public anonymous persona bound to an internal user.
type AnonymousProfile struct {
	UserID         string    `xorm:"not null pk BIGINT(20) user_id"`
	CreatedAt      time.Time `xorm:"created TIMESTAMP created_at"`
	UpdatedAt      time.Time `xorm:"updated TIMESTAMP updated_at"`
	AnonSubjectID  string    `xorm:"not null unique VARCHAR(64) anon_subject_id"`
	DisplayName    string    `xorm:"not null default '' VARCHAR(64) display_name"`
	Avatar         string    `xorm:"not null default '' VARCHAR(2048) avatar"`
	AvatarSeed     string    `xorm:"not null default '' VARCHAR(64) avatar_seed"`
	Status         string    `xorm:"not null default 'active' VARCHAR(20) status"`
	RevealDisabled bool      `xorm:"not null default false BOOL reveal_disabled"`
}

func (AnonymousProfile) TableName() string {
	return "anonymous_profile"
}
