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

import useSWR from 'swr';
import qs from 'qs';

import request from '@/utils/request';
import type { ListResult } from '@/common/interface';

export interface CommunityAnonymousInfo {
  enabled: boolean;
  anon_subject_id: string;
  display_name: string;
  avatar_seed: string;
}

export interface CommunityActor {
  id: string;
  username?: string;
  rank?: number;
  display_name?: string;
  avatar?: string;
  status?: string;
  anonymous?: CommunityAnonymousInfo;
}

export interface CommunityTag {
  slug_name: string;
  display_name: string;
  recommend?: boolean;
  reserved?: boolean;
}

export interface CommunityFeedItem {
  id: string;
  created_at: number;
  title: string;
  description: string;
  channel_type: 'qa' | 'discussion';
  visibility_mode: string;
  moderation_state: string;
  tags: CommunityTag[];
  view_count: number;
  vote_count: number;
  answer_count: number;
  follow_count: number;
  pin: number;
  show: number;
  status: number;
  operated_at: number;
  operation_type: string;
  operator: CommunityActor;
}

export interface CommunityQuestionDetail {
  id: string;
  title: string;
  content: string;
  html: string;
  tags: CommunityTag[];
  create_time: number;
  update_time: number;
  answer_count: number;
  view_count: number;
  vote_count: number;
  channel_type: 'qa' | 'discussion';
  visibility_mode: string;
  moderation_state: string;
  status: number;
  show: number;
  user_info?: CommunityActor;
  update_user_info?: CommunityActor;
  last_answered_user_info?: CommunityActor;
}

export interface CommunityReply {
  id: string;
  question_id: string;
  html: string;
  content: string;
  create_time: number;
  update_time: number;
  vote_count: number;
  status: number;
  user_info?: CommunityActor;
  update_user_info?: CommunityActor;
}

export interface CommunityComment {
  comment_id: string;
  created_at: number;
  object_id: string;
  parsed_text: string;
  original_text: string;
  user_display_name: string;
  reply_user_display_name?: string;
  user_anonymous?: CommunityAnonymousInfo;
  reply_user_anonymous?: CommunityAnonymousInfo;
}

export interface CommunityDetailResp {
  question: CommunityQuestionDetail;
  replies: CommunityReply[];
  reply_count: number;
}

export interface CommunityPreviewBootstrapResp {
  enabled: boolean;
  mode: string;
  seeded: boolean;
}

export interface CommunityPreviewLoginResp {
  access_token: string;
  redirect_url: string;
  anon_subject_id: string;
  display_name: string;
  avatar_seed: string;
}

export interface CommunityFeedParams {
  page?: number;
  page_size?: number;
  order?: string;
  channel?: 'qa' | 'discussion';
  include_all?: boolean;
}

export const useCommunityFeed = (params: CommunityFeedParams) => {
  const apiUrl = `/answer/api/v1/home?${qs.stringify(params, {
    skipNulls: true,
  })}`;
  const { data, error, mutate } = useSWR<ListResult<CommunityFeedItem>, Error>(
    apiUrl,
    request.instance.get,
  );
  return {
    data,
    isLoading: !data && !error,
    error,
    mutate,
  };
};

export const useCommunityDetail = (
  channel: 'qa' | 'discussion',
  id?: string,
) => {
  const apiUrl =
    channel === 'qa'
      ? `/answer/api/v1/questions/${id}`
      : `/answer/api/v1/discussions/${id}`;
  const { data, error, mutate } = useSWR<CommunityDetailResp, Error>(
    id ? apiUrl : null,
    request.instance.get,
  );
  return {
    data,
    isLoading: !data && !error,
    error,
    mutate,
  };
};

export const createCommunityPost = (
  channel: 'qa' | 'discussion',
  params: Record<string, any>,
) => {
  const apiUrl =
    channel === 'qa'
      ? '/answer/api/v1/questions'
      : '/answer/api/v1/discussions';
  return request.post(apiUrl, params);
};

export const createCommunityReply = (
  questionId: string,
  params: Record<string, any>,
) => request.post(`/answer/api/v1/content/${questionId}/replies`, params);

export const useCommunityReplyComments = (
  answerId: string,
  params: { page?: number; page_size?: number } = {},
) => {
  const apiUrl = `/answer/api/v1/replies/${answerId}/comments?${qs.stringify(
    {
      page: 1,
      page_size: 20,
      ...params,
    },
    { skipNulls: true },
  )}`;
  const { data, error, mutate } = useSWR<ListResult<CommunityComment>, Error>(
    answerId ? apiUrl : null,
    request.instance.get,
  );
  return {
    data,
    isLoading: !data && !error,
    error,
    mutate,
  };
};

export const createCommunityComment = (
  answerId: string,
  params: Record<string, any>,
) => request.post(`/answer/api/v1/replies/${answerId}/comments`, params);

export const bootstrapCommunityPreview = () =>
  request.get<CommunityPreviewBootstrapResp>(
    '/answer/api/v1/community/preview/bootstrap',
  );

export const previewCommunityLogin = () =>
  request.post<CommunityPreviewLoginResp>(
    '/answer/api/v1/community/preview/login',
    {},
  );

export const communityModerate = (params: {
  object_type: 'question' | 'answer' | 'comment';
  object_id: string;
  action: 'hide' | 'unhide' | 'delete' | 'restore';
  reason?: string;
}) => request.post('/answer/api/v1/admin/moderation/actions', params);
