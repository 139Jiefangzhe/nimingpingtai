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

import { FC, useState } from 'react';
import { Button, Card, Col, Form, Row } from 'react-bootstrap';
import { useLocation, useParams } from 'react-router-dom';

import { Tag } from '@/components';
import { usePageTags } from '@/hooks';
import { loggedUserInfoStore, toastStore } from '@/stores';
import {
  createCommunityComment,
  createCommunityReply,
  useCommunityDetail,
  useCommunityReplyComments,
} from '@/services/client/community';
import {
  actorDisplayName,
  CommunityAnonAvatar,
  CommunityChannelBadge,
  CommunityModerationBadge,
} from '../shared';

const ReplyComments: FC<{ answerId: string; canComment: boolean }> = ({
  answerId,
  canComment,
}) => {
  const { data, mutate } = useCommunityReplyComments(answerId, {
    page: 1,
    page_size: 20,
  });
  const [draft, setDraft] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    if (!draft.trim()) {
      return;
    }
    try {
      setSubmitting(true);
      await createCommunityComment(answerId, {
        original_text: draft.trim(),
      });
      setDraft('');
      mutate();
      toastStore.getState().show({ msg: '评论已发布', variant: 'success' });
    } catch (error: any) {
      toastStore.getState().show({
        msg: error?.message || '评论发布失败',
        variant: 'danger',
      });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="mt-3">
      <div className="d-flex flex-column gap-2">
        {data?.list?.map((comment) => (
          <div
            key={comment.comment_id}
            className="border rounded-4 p-3 bg-light">
            <div className="d-flex align-items-center gap-3 mb-2">
              <CommunityAnonAvatar actor={comment.user_anonymous} size={32} />
              <div>
                <div className="fw-semibold">
                  {comment.user_anonymous?.display_name ||
                    comment.user_display_name}
                </div>
                <div className="small text-secondary">
                  {new Date(comment.created_at * 1000).toLocaleString()}
                </div>
              </div>
            </div>
            <div
              className="community-html fmt small"
              dangerouslySetInnerHTML={{ __html: comment.parsed_text }}
            />
          </div>
        ))}
      </div>

      {canComment ? (
        <div className="mt-3">
          <Form.Control
            as="textarea"
            rows={2}
            placeholder="写下你的评论..."
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
          />
          <div className="mt-2 d-flex justify-content-end">
            <Button
              size="sm"
              onClick={handleSubmit}
              disabled={submitting || !draft.trim()}>
              {submitting ? '发布中...' : '发布评论'}
            </Button>
          </div>
        </div>
      ) : (
        <div className="small text-secondary mt-3">
          先在顶部进入匿名社区，才能评论。
        </div>
      )}
    </div>
  );
};

const Detail: FC = () => {
  const { id = '' } = useParams();
  const location = useLocation();
  const channel = location.pathname.includes('/community/questions/')
    ? 'qa'
    : 'discussion';
  const loggedUser = loggedUserInfoStore((state) => state.user);
  const canReply = Boolean(loggedUser?.access_token);
  const { data, isLoading, mutate } = useCommunityDetail(channel, id);
  const [replyContent, setReplyContent] = useState('');
  const [submitting, setSubmitting] = useState(false);

  usePageTags({
    title: data?.question?.title || '匿名社区详情',
  });

  const handleReply = async () => {
    if (!replyContent.trim() || !data?.question?.id) {
      return;
    }
    try {
      setSubmitting(true);
      await createCommunityReply(data.question.id, {
        content: replyContent.trim(),
      });
      setReplyContent('');
      mutate();
      toastStore.getState().show({ msg: '回复已发布', variant: 'success' });
    } catch (error: any) {
      toastStore.getState().show({
        msg: error?.message || '回复发布失败',
        variant: 'danger',
      });
    } finally {
      setSubmitting(false);
    }
  };

  if (isLoading) {
    return (
      <Card className="community-card border-0">
        <Card.Body className="py-5 text-center text-secondary">
          正在加载帖子详情...
        </Card.Body>
      </Card>
    );
  }

  if (!data?.question) {
    return (
      <Card className="community-card border-0">
        <Card.Body className="py-5 text-center text-secondary">
          这条内容不存在，或者已经被处理。
        </Card.Body>
      </Card>
    );
  }

  return (
    <Row className="g-4 pb-4">
      <Col xl={8}>
        <Card className="community-card border-0 mb-4">
          <Card.Body className="p-4 p-lg-5">
            <div className="d-flex flex-wrap gap-2 mb-3">
              <CommunityChannelBadge channel={data.question.channel_type} />
              <CommunityModerationBadge
                show={data.question.show}
                status={data.question.status}
                moderationState={data.question.moderation_state}
              />
            </div>

            <h2 className="mb-4">{data.question.title}</h2>

            <div className="d-flex align-items-center gap-3 mb-4">
              <CommunityAnonAvatar actor={data.question.user_info} size={44} />
              <div>
                <div className="fw-semibold">
                  {actorDisplayName(data.question.user_info)}
                </div>
                <div className="small text-secondary">
                  {new Date(data.question.create_time * 1000).toLocaleString()}
                </div>
              </div>
            </div>

            <div
              className="community-html fmt"
              dangerouslySetInnerHTML={{ __html: data.question.html }}
            />

            {!!data.question.tags?.length && (
              <div className="community-tags mt-4">
                {data.question.tags.map((tag) => (
                  <Tag
                    key={`${data.question.id}-${tag.slug_name}`}
                    data={{
                      slug_name: tag.slug_name,
                      display_name: tag.display_name,
                    }}
                  />
                ))}
              </div>
            )}
          </Card.Body>
        </Card>

        <div className="mb-3 d-flex justify-content-between align-items-center">
          <h4 className="mb-0">回复 {data.reply_count}</h4>
          <div className="small text-secondary">
            {channel === 'qa' ? '更适合结构化回答' : '更适合顺序讨论'}
          </div>
        </div>

        <div className="d-flex flex-column gap-3">
          {data.replies?.map((reply) => (
            <Card key={reply.id} className="community-card border-0">
              <Card.Body className="p-4">
                <div className="d-flex align-items-center gap-3 mb-3">
                  <CommunityAnonAvatar actor={reply.user_info} size={40} />
                  <div>
                    <div className="fw-semibold">
                      {actorDisplayName(reply.user_info)}
                    </div>
                    <div className="small text-secondary">
                      {new Date(reply.create_time * 1000).toLocaleString()}
                    </div>
                  </div>
                </div>

                <div
                  className="community-html fmt"
                  dangerouslySetInnerHTML={{ __html: reply.html }}
                />

                <div className="small text-secondary mt-3">
                  支持票数：{reply.vote_count || 0}
                </div>

                <ReplyComments answerId={reply.id} canComment={canReply} />
              </Card.Body>
            </Card>
          ))}
        </div>

        <Card className="community-card border-0 mt-4">
          <Card.Body className="p-4 community-reply-box">
            <div className="d-flex justify-content-between align-items-center mb-3">
              <h5 className="mb-0">
                {channel === 'qa' ? '补充回答' : '参与讨论'}
              </h5>
              {!canReply && (
                <span className="small text-secondary">需要先登录</span>
              )}
            </div>
            <Form.Control
              as="textarea"
              rows={5}
              placeholder={
                channel === 'qa' ? '写下你的回答...' : '写下你的回复...'
              }
              value={replyContent}
              onChange={(event) => setReplyContent(event.target.value)}
              disabled={!canReply}
            />
            <div className="mt-3 d-flex justify-content-end">
              <Button
                onClick={handleReply}
                disabled={!canReply || submitting || !replyContent.trim()}>
                {submitting ? '发布中...' : '发布回复'}
              </Button>
            </div>
          </Card.Body>
        </Card>
      </Col>

      <Col xl={4}>
        <div className="community-sidebar-card d-flex flex-column gap-3">
          <Card className="community-card border-0">
            <Card.Body>
              <div className="text-secondary small mb-2">帖子概览</div>
              <div className="community-meta flex-column align-items-start gap-2">
                <span>查看：{data.question.view_count}</span>
                <span>回复：{data.question.answer_count}</span>
                <span>频道：{channel === 'qa' ? '匿名问答' : '匿名交流'}</span>
              </div>
            </Card.Body>
          </Card>
          <Card className="community-card border-0">
            <Card.Body>
              <div className="text-secondary small mb-2">当前作者</div>
              <div className="d-flex align-items-center gap-3">
                <CommunityAnonAvatar
                  actor={data.question.user_info}
                  size={48}
                />
                <div>
                  <div className="fw-semibold">
                    {actorDisplayName(data.question.user_info)}
                  </div>
                  <div className="small text-secondary">前台只展示匿名身份</div>
                </div>
              </div>
            </Card.Body>
          </Card>
        </div>
      </Col>
    </Row>
  );
};

export default Detail;
