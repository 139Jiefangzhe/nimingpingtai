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

import { FC } from 'react';
import { Badge, Button, Card, Col, Row } from 'react-bootstrap';
import { Link, useLocation, useSearchParams } from 'react-router-dom';

import { Pagination, Tag } from '@/components';
import { usePageTags } from '@/hooks';
import { loggedUserInfoStore } from '@/stores';
import { useCommunityFeed } from '@/services/client/community';
import {
  actorDisplayName,
  CommunityAnonAvatar,
  CommunityChannelBadge,
  CommunityModerationBadge,
} from '../shared';

const orderLabels: Record<string, string> = {
  newest: '最新',
  active: '活跃',
  hot: '热门',
};

const Feed: FC = () => {
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const loggedUser = loggedUserInfoStore((state) => state.user);
  const channel = location.pathname.includes('/community/qa')
    ? 'qa'
    : 'discussion';
  const page = Number(searchParams.get('page') || 1);
  const order = searchParams.get('order') || 'newest';
  const { data, isLoading } = useCommunityFeed({
    channel,
    page,
    page_size: 10,
    order,
  });

  usePageTags({
    title: channel === 'qa' ? '匿名问答' : '匿名交流',
    subtitle:
      channel === 'qa' ? '面向问题沉淀的内部问答区' : '更轻量的匿名讨论频道',
  });

  const detailBase =
    channel === 'qa' ? '/community/questions' : '/community/discussions';

  return (
    <Row className="g-4 pb-4">
      <Col xl={8}>
        <div className="d-flex flex-wrap justify-content-between align-items-center gap-3 mb-3">
          <div>
            <div className="text-secondary small">
              {channel === 'qa' ? '面向可沉淀问题' : '更适合观点和交流'}
            </div>
            <h2 className="mb-0">
              {channel === 'qa' ? '匿名问答' : '匿名交流'}
            </h2>
          </div>
          <div className="d-flex flex-wrap gap-2">
            {Object.entries(orderLabels).map(([key, label]) => {
              const current = searchParams.toString();
              const next = new URLSearchParams(current);
              next.set('order', key);
              next.set('page', '1');
              return (
                <Button
                  as={Link as any}
                  key={key}
                  to={`${location.pathname}?${next.toString()}`}
                  variant={order === key ? 'primary' : 'outline-secondary'}
                  size="sm">
                  {label}
                </Button>
              );
            })}
          </div>
        </div>

        {isLoading && (
          <Card className="community-card border-0">
            <Card.Body className="py-5 text-center text-secondary">
              正在加载社区内容...
            </Card.Body>
          </Card>
        )}

        {!isLoading && !data?.list?.length && (
          <Card className="community-card border-0">
            <Card.Body className="py-5 text-center text-secondary">
              还没有内容，先发第一条匿名帖子。
            </Card.Body>
          </Card>
        )}

        <div className="d-flex flex-column gap-3">
          {data?.list?.map((item) => (
            <Card key={item.id} className="community-card border-0">
              <Card.Body className="p-4">
                <div className="d-flex justify-content-between align-items-start gap-3 mb-3">
                  <div className="d-flex gap-2 flex-wrap">
                    <CommunityChannelBadge channel={item.channel_type} />
                    <CommunityModerationBadge
                      show={item.show}
                      status={item.status}
                      moderationState={item.moderation_state}
                    />
                    {item.pin === 2 && <Badge bg="dark">置顶</Badge>}
                  </div>
                  <div className="community-meta">
                    <span>{item.answer_count} 条回复</span>
                    <span>{item.view_count} 次查看</span>
                    <span>{item.vote_count} 票支持</span>
                  </div>
                </div>

                <Link
                  to={`${detailBase}/${item.id}`}
                  className="community-card-title">
                  <h4 className="mb-3">{item.title}</h4>
                </Link>

                <p className="text-secondary mb-3">{item.description}</p>

                {!!item.tags?.length && (
                  <div className="community-tags mb-3">
                    {item.tags.map((tag) => (
                      <Tag
                        key={`${item.id}-${tag.slug_name}`}
                        data={{
                          slug_name: tag.slug_name,
                          display_name: tag.display_name,
                          recommend: tag.recommend,
                          reserved: tag.reserved,
                        }}
                      />
                    ))}
                  </div>
                )}

                <div className="d-flex justify-content-between align-items-center gap-3 flex-wrap">
                  <div className="d-flex align-items-center gap-3">
                    <CommunityAnonAvatar actor={item.operator} />
                    <div>
                      <div className="fw-semibold">
                        {actorDisplayName(item.operator)}
                      </div>
                      <div className="small text-secondary">
                        {item.operation_type === 'answered'
                          ? '最近回复'
                          : item.operation_type === 'modified'
                            ? '最近编辑'
                            : '发起讨论'}
                      </div>
                    </div>
                  </div>
                  <div className="small text-secondary">
                    {new Date(item.operated_at * 1000).toLocaleString()}
                  </div>
                </div>
              </Card.Body>
            </Card>
          ))}
        </div>

        <div className="mt-4">
          <Pagination
            currentPage={page}
            pageSize={10}
            totalSize={data?.count || 0}
            pathname={location.pathname}
          />
        </div>
      </Col>

      <Col xl={4}>
        <div className="community-sidebar-card d-flex flex-column gap-3">
          <Card className="community-card border-0">
            <Card.Body>
              <div className="text-secondary small mb-2">发帖入口</div>
              <h5 className="mb-2">
                {channel === 'qa' ? '沉淀高质量问答' : '先把真实意见说出来'}
              </h5>
              <p className="text-secondary mb-3">
                {channel === 'qa'
                  ? '适合整理明确问题、方法和结论。'
                  : '适合发起观点征集、匿名讨论和团队温度话题。'}
              </p>
              {loggedUser?.access_token ? (
                <Button
                  as={Link as any}
                  to={
                    channel === 'qa'
                      ? '/community/questions/new'
                      : '/community/discussions/new'
                  }
                  className="w-100">
                  {channel === 'qa' ? '发布问答' : '发布讨论'}
                </Button>
              ) : (
                <div className="small text-secondary">
                  先在顶部点击“进入匿名社区”，再发帖。
                </div>
              )}
            </Card.Body>
          </Card>

          <Card className="community-card border-0">
            <Card.Body>
              <div className="text-secondary small mb-2">当前模式</div>
              <ul className="mb-0 text-secondary ps-3">
                <li>前台以匿名身份展示作者</li>
                <li>问答和讨论拆成两个频道</li>
                <li>本地预览自动带演示帖子和回复</li>
              </ul>
            </Card.Body>
          </Card>
        </div>
      </Col>
    </Row>
  );
};

export default Feed;
