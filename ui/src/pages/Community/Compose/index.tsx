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

import { FC, useMemo, useState } from 'react';
import { Button, Card, Col, Form, Row } from 'react-bootstrap';
import { useLocation, useNavigate } from 'react-router-dom';

import { usePageTags } from '@/hooks';
import { toastStore } from '@/stores';
import { createCommunityPost } from '@/services/client/community';

const parseTags = (raw: string) => {
  return raw
    .split(/[，,]/)
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => ({
      slug_name: item.toLowerCase().replace(/\s+/g, '-'),
      display_name: item,
      original_text: item,
    }));
};

const Compose: FC = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const channel = location.pathname.includes('/community/questions/new')
    ? 'qa'
    : 'discussion';
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [tags, setTags] = useState(channel === 'qa' ? 'Culture, Workflow' : '');
  const [submitting, setSubmitting] = useState(false);

  const pageMeta = useMemo(() => {
    return channel === 'qa'
      ? {
          title: '发布匿名问答',
          subtitle: '适合提明确问题，并沉淀成FAQ',
        }
      : {
          title: '发布匿名讨论',
          subtitle: '适合发观点、提建议、做征集',
        };
  }, [channel]);

  usePageTags(pageMeta);

  const handleSubmit = async () => {
    if (!content.trim() || (channel === 'qa' && !title.trim())) {
      return;
    }
    try {
      setSubmitting(true);
      const payload =
        channel === 'qa'
          ? {
              title: title.trim(),
              content: content.trim(),
              tags: parseTags(tags),
            }
          : {
              title: title.trim(),
              content: content.trim(),
              tags: parseTags(tags),
            };
      const resp = await createCommunityPost(channel, payload);
      toastStore.getState().show({
        msg: channel === 'qa' ? '问答已发布' : '讨论已发布',
        variant: 'success',
      });
      navigate(
        channel === 'qa'
          ? `/community/questions/${resp.id}`
          : `/community/discussions/${resp.id}`,
      );
    } catch (error: any) {
      toastStore.getState().show({
        msg: error?.message || '发布失败',
        variant: 'danger',
      });
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Row className="g-4 pb-4">
      <Col xl={8}>
        <Card className="community-card border-0">
          <Card.Body className="p-4 p-lg-5 community-composer">
            <div className="text-secondary small mb-2">{pageMeta.subtitle}</div>
            <h2 className="mb-4">{pageMeta.title}</h2>

            <Form.Group className="mb-3">
              <Form.Label>
                {channel === 'qa' ? '标题' : '标题（可选）'}
              </Form.Label>
              <Form.Control
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                placeholder={
                  channel === 'qa'
                    ? '写一个清晰、具体的问题标题'
                    : '如果留空，系统会从正文自动提取'
                }
              />
            </Form.Group>

            <Form.Group className="mb-3">
              <Form.Label>正文</Form.Label>
              <Form.Control
                as="textarea"
                value={content}
                onChange={(event) => setContent(event.target.value)}
                placeholder={
                  channel === 'qa'
                    ? '补充问题背景、现状和你想得到的答案'
                    : '直接写下你真正想说的话'
                }
              />
            </Form.Group>

            <Form.Group className="mb-4">
              <Form.Label>
                {channel === 'qa' ? '标签' : '标签（可选）'}
              </Form.Label>
              <Form.Control
                value={tags}
                onChange={(event) => setTags(event.target.value)}
                placeholder="多个标签用英文逗号分隔"
              />
              <Form.Text className="text-secondary">
                示例：Culture, Workflow, Product
              </Form.Text>
            </Form.Group>

            <div className="d-flex justify-content-end">
              <Button
                onClick={handleSubmit}
                disabled={
                  submitting ||
                  !content.trim() ||
                  (channel === 'qa' && !title.trim())
                }>
                {submitting ? '发布中...' : '立即发布'}
              </Button>
            </div>
          </Card.Body>
        </Card>
      </Col>

      <Col xl={4}>
        <div className="community-sidebar-card d-flex flex-column gap-3">
          <Card className="community-card border-0">
            <Card.Body>
              <div className="text-secondary small mb-2">写作建议</div>
              <ul className="mb-0 text-secondary ps-3">
                <li>先写结论，再补背景。</li>
                <li>避免暴露敏感项目、人名和客户信息。</li>
                <li>匿名不等于免审，管理端仍可处理违规内容。</li>
              </ul>
            </Card.Body>
          </Card>
        </div>
      </Col>
    </Row>
  );
};

export default Compose;
