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
import { Badge } from 'react-bootstrap';

import type {
  CommunityActor,
  CommunityAnonymousInfo,
} from '@/services/client/community';

export const actorDisplayName = (
  actor?: CommunityActor | CommunityAnonymousInfo | null,
) => {
  if (!actor) {
    return '匿名成员';
  }
  if ('anonymous' in actor && actor.anonymous?.display_name) {
    return actor.anonymous.display_name;
  }
  if ('display_name' in actor && actor.display_name) {
    return actor.display_name;
  }
  return '匿名成员';
};

export const actorAvatarSeed = (
  actor?: CommunityActor | CommunityAnonymousInfo | null,
) => {
  if (!actor) {
    return 'community';
  }
  if ('anonymous' in actor && actor.anonymous?.avatar_seed) {
    return actor.anonymous.avatar_seed;
  }
  if ('avatar_seed' in actor && actor.avatar_seed) {
    return actor.avatar_seed;
  }
  if ('display_name' in actor && actor.display_name) {
    return actor.display_name;
  }
  return 'community';
};

const hashHue = (seed: string) => {
  let value = 0;
  for (let i = 0; i < seed.length; i += 1) {
    value = (value * 31 + seed.charCodeAt(i)) % 360;
  }
  return value;
};

export const CommunityAnonAvatar: FC<{
  actor?: CommunityActor | CommunityAnonymousInfo | null;
  size?: number;
}> = ({ actor, size = 40 }) => {
  const label = actorDisplayName(actor);
  const seed = actorAvatarSeed(actor);
  const hue = hashHue(seed);
  const text =
    label.length <= 2
      ? label
      : label.replace(/\s+/g, '').slice(label.length - 2);

  return (
    <div
      className="community-anon-avatar rounded-circle d-inline-flex align-items-center justify-content-center fw-semibold"
      style={{
        width: size,
        height: size,
        background: `linear-gradient(135deg, hsl(${hue} 72% 82%), hsl(${
          (hue + 48) % 360
        } 63% 64%))`,
      }}>
      <span>{text}</span>
    </div>
  );
};

export const CommunityChannelBadge: FC<{ channel: 'qa' | 'discussion' }> = ({
  channel,
}) => {
  return (
    <Badge
      bg={channel === 'qa' ? 'primary' : 'warning'}
      text={channel === 'qa' ? undefined : 'dark'}>
      {channel === 'qa' ? '问答' : '讨论'}
    </Badge>
  );
};

export const CommunityModerationBadge: FC<{
  show?: number;
  status?: number;
  moderationState?: string;
}> = ({ show = 0, status = 1, moderationState = 'normal' }) => {
  if (status === 10) {
    return <Badge bg="secondary">已删除</Badge>;
  }
  if (show === 2 || moderationState === 'blocked') {
    return <Badge bg="danger">已隐藏</Badge>;
  }
  if (status === 11 || moderationState === 'pending') {
    return <Badge bg="info">待审核</Badge>;
  }
  return <Badge bg="success">正常</Badge>;
};
