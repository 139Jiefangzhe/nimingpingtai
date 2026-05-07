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
import { Button, Card, Col, Row } from 'react-bootstrap';
import { Link } from 'react-router-dom';

import { usePageTags } from '@/hooks';
import { toastStore } from '@/stores';
import {
  communityModerate,
  useCommunityFeed,
} from '@/services/client/community';
import {
  actorDisplayName,
  CommunityAnonAvatar,
  CommunityModerationBadge,
} from '../shared';

const ModerationSection: FC<{
  title: string;
  channel: 'qa' | 'discussion';
}> = ({ title, channel }) => {
  const { data, mutate } = useCommunityFeed({
    channel,
    page: 1,
    page_size: 20,
    order: 'newest',
    include_all: true,
  });

  const handleAction = async (
    objectId: string,
    action: 'hide' | 'unhide' | 'delete' | 'restore',
  ) => {
    try {
      await communityModerate({
        object_type: 'question',
        object_id: objectId,
        action,
        reason: '本地预览手动操作',
      });
      mutate();
      toastStore.getState().show({ msg: '处理完成', variant: 'success' });
    } catch (error: any) {
      toastStore.getState().show({
        msg: error?.message || '处理失败',
        variant: 'danger',
      });
    }
  };

  return (
    <Card className="community-card border-0">
      <Card.Body className="p-4">
        <div className="d-flex justify-content-between align-items-center mb-3">
          <h4 className="mb-0">{title}</h4>
          <div className="small text-secondary">
            共 {data?.count || 0} 条内容
          </div>
        </div>

        <div className="d-flex flex-column gap-3">
          {data?.list?.map((item) => (
            <div key={item.id} className="border rounded-4 p-3">
              <div className="d-flex justify-content-between align-items-start gap-3 mb-3 flex-wrap">
                <div>
                  <div className="fw-semibold mb-1">{item.title}</div>
                  <div className="small text-secondary">
                    {new Date(item.operated_at * 1000).toLocaleString()}
                  </div>
                </div>
                <CommunityModerationBadge
                  show={item.show}
                  status={item.status}
                  moderationState={item.moderation_state}
                />
              </div>

              <div className="d-flex align-items-center gap-3 mb-3">
                <CommunityAnonAvatar actor={item.operator} size={36} />
                <div className="small text-secondary">
                  {actorDisplayName(item.operator)}
                </div>
              </div>

              <div className="d-flex flex-wrap gap-2">
                <Button
                  as={Link as any}
                  to={
                    channel === 'qa'
                      ? `/community/questions/${item.id}`
                      : `/community/discussions/${item.id}`
                  }
                  size="sm"
                  variant="outline-secondary">
                  查看
                </Button>
                {item.status === 10 ? (
                  <Button
                    size="sm"
                    variant="success"
                    onClick={() => handleAction(item.id, 'restore')}>
                    恢复
                  </Button>
                ) : (
                  <>
                    <Button
                      size="sm"
                      variant={item.show === 2 ? 'success' : 'warning'}
                      onClick={() =>
                        handleAction(
                          item.id,
                          item.show === 2 ? 'unhide' : 'hide',
                        )
                      }>
                      {item.show === 2 ? '取消隐藏' : '隐藏'}
                    </Button>
                    <Button
                      size="sm"
                      variant="danger"
                      onClick={() => handleAction(item.id, 'delete')}>
                      删除
                    </Button>
                  </>
                )}
              </div>
            </div>
          ))}
        </div>
      </Card.Body>
    </Card>
  );
};

const Moderation: FC = () => {
  usePageTags({
    title: '社区管理',
    subtitle: '本地预览审核台',
  });

  return (
    <Row className="g-4 pb-4">
      <Col xl={6}>
        <ModerationSection title="讨论频道管理" channel="discussion" />
      </Col>
      <Col xl={6}>
        <ModerationSection title="问答频道管理" channel="qa" />
      </Col>
    </Row>
  );
};

export default Moderation;
