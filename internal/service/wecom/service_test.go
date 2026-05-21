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

package wecom

import "testing"

func TestIsLinkedCorpUserID(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		want   bool
	}{
		{name: "internal member", userID: "XYBH-0038", want: false},
		{name: "linked corp member", userID: "wwa07100652cfdffd5/panyu", want: true},
		{name: "missing suffix", userID: "wwa07100652cfdffd5/", want: false},
		{name: "missing prefix", userID: "/panyu", want: false},
		{name: "empty", userID: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLinkedCorpUserID(tt.userID); got != tt.want {
				t.Fatalf("isLinkedCorpUserID(%q) = %v, want %v", tt.userID, got, tt.want)
			}
		})
	}
}

func TestJoinDepartmentIntIDs(t *testing.T) {
	tests := []struct {
		name string
		ids  []int
		want string
	}{
		{name: "empty", ids: nil, want: ""},
		{name: "single", ids: []int{12}, want: "12"},
		{name: "multiple", ids: []int{36, 12, 99}, want: "36,12,99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := joinDepartmentIntIDs(tt.ids); got != tt.want {
				t.Fatalf("joinDepartmentIntIDs(%v) = %q, want %q", tt.ids, got, tt.want)
			}
		})
	}
}
