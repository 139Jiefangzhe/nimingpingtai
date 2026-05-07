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

import { FC, memo } from 'react';
import { NavLink, Outlet, useLocation } from 'react-router-dom';
import { Button } from 'react-bootstrap';

import { Footer } from '@/components';
import { usePageTags } from '@/hooks';
import { loggedUserInfoStore } from '@/stores';
import Storage from '@/utils/storage';
import { REDIRECT_PATH_STORAGE_KEY } from '@/common/constants';

import './index.scss';

const Layout: FC = () => {
  const location = useLocation();
  const user = loggedUserInfoStore((state) => state.user);

  usePageTags({
    title: '匿名社区',
    subtitle: '企业微信登录版',
  });

  const handleWeComEntry = () => {
    Storage.set(
      REDIRECT_PATH_STORAGE_KEY,
      `${location.pathname}${location.search}` || '/community',
    );
    window.location.href =
      '/answer/api/v1/wecom/auth/start?return_to=' +
      encodeURIComponent(location.pathname + location.search);
  };

  if (!user?.access_token) {
    return (
      <div className="community-shell">
        <div className="container-xxl py-4">
          <section className="community-hero card border-0 shadow-sm overflow-hidden">
            <div className="card-body p-4 p-lg-5">
              <div className="d-flex flex-column flex-lg-row justify-content-between gap-4">
                <div>
                  <h1 className="community-hero-title mb-3">匿名交流社区</h1>
                  <p className="text-secondary mb-0 community-hero-copy">
                    匿名提问、匿名讨论、回复评论。请通过企业微信登录后访问。
                  </p>
                </div>
                <div className="community-hero-side">
                  <Button className="w-100" onClick={handleWeComEntry}>
                    企业微信登录
                  </Button>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>
    );
  }

  return (
    <div className="community-shell">
      <div className="container-xxl py-4">
        <section className="community-hero card border-0 shadow-sm overflow-hidden">
          <div className="card-body p-4 p-lg-5">
            <div className="d-flex flex-column flex-lg-row justify-content-between gap-4">
              <div>
                <h1 className="community-hero-title mb-3">匿名交流社区</h1>
                <p className="text-secondary mb-0 community-hero-copy">
                  匿名提问、匿名讨论、回复评论。当前会话：
                  {user.display_name || user.username}
                </p>
              </div>
            </div>
          </div>
        </section>

        <nav className="community-nav mt-4 mb-4">
          <NavLink end to="/community" className="community-nav-link">
            交流
          </NavLink>
          <NavLink to="/community/qa" className="community-nav-link">
            问答
          </NavLink>
          {user.access_token && (
            <>
              <NavLink
                to="/community/discussions/new"
                className="community-nav-link">
                发讨论
              </NavLink>
              <NavLink
                to="/community/questions/new"
                className="community-nav-link">
                发问答
              </NavLink>
            </>
          )}
          {(user.role_id === 2 || user.role_id === 3) && (
            <NavLink to="/community/moderation" className="community-nav-link">
              管理
            </NavLink>
          )}
        </nav>

        <Outlet />
      </div>
      <div className="container-xxl pb-4">
        <Footer />
      </div>
    </div>
  );
};

export default memo(Layout);
