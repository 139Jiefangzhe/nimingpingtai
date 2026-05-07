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

import { FC, memo, useEffect, useState } from 'react';
import { NavLink, Outlet, useLocation } from 'react-router-dom';
import { Button, Spinner } from 'react-bootstrap';

import { Footer } from '@/components';
import { usePageTags } from '@/hooks';
import { loggedUserInfoStore, toastStore } from '@/stores';
import {
  bootstrapCommunityPreview,
  previewCommunityLogin,
} from '@/services/client/community';
import Storage from '@/utils/storage';
import { REDIRECT_PATH_STORAGE_KEY } from '@/common/constants';

import './index.scss';

const Layout: FC = () => {
  const location = useLocation();
  const user = loggedUserInfoStore((state) => state.user);
  const [loading, setLoading] = useState(true);
  const [previewMode, setPreviewMode] = useState('');
  const [bootError, setBootError] = useState('');
  const [isEntering, setIsEntering] = useState(false);

  usePageTags({
    title: '匿名社区',
    subtitle: '本地预览版',
  });

  useEffect(() => {
    let active = true;
    bootstrapCommunityPreview()
      .then((resp) => {
        if (!active) {
          return;
        }
        setPreviewMode(resp?.mode || '');
      })
      .catch((error) => {
        if (!active) {
          return;
        }
        setBootError(error?.message || '社区预览初始化失败');
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, []);

  const handlePreviewEntry = async () => {
    try {
      setIsEntering(true);
      Storage.set(
        REDIRECT_PATH_STORAGE_KEY,
        `${location.pathname}${location.search}` || '/community',
      );
      const resp = await previewCommunityLogin();
      if (resp?.redirect_url) {
        window.location.assign(resp.redirect_url);
        return;
      }
      toastStore
        .getState()
        .show({ msg: '匿名社区进入失败', variant: 'danger' });
    } catch (error: any) {
      toastStore.getState().show({
        msg: error?.message || '匿名社区进入失败',
        variant: 'danger',
      });
    } finally {
      setIsEntering(false);
    }
  };

  if (loading) {
    return (
      <div className="container-xxl py-5">
        <div className="card border-0 shadow-sm">
          <div className="card-body py-5 text-center">
            <Spinner animation="border" />
            <div className="mt-3 text-secondary">
              正在准备匿名社区预览环境...
            </div>
          </div>
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
                <div className="text-uppercase small fw-semibold text-secondary mb-2">
                  Anonymous Community Preview
                </div>
                <h1 className="community-hero-title mb-3">匿名交流社区</h1>
                <p className="text-secondary mb-0 community-hero-copy">
                  当前是本机 Docker
                  预览版。这里优先演示匿名问答、匿名讨论、回复评论和基础管理壳子，不依赖企业微信登录。
                </p>
              </div>
              <div className="community-hero-side">
                <div className="small text-secondary mb-2">
                  预览模式：{previewMode || 'local'}
                </div>
                <div className="community-session-card">
                  <div className="small text-secondary mb-1">当前会话</div>
                  <div className="fw-semibold">
                    {user?.access_token
                      ? user.display_name || user.username
                      : '未登录'}
                  </div>
                  <div className="small text-secondary mt-1">
                    {user?.access_token
                      ? user.role_id === 2
                        ? '管理员'
                        : user.role_id === 3
                          ? '版主'
                          : '匿名访客会话'
                      : '点击按钮进入演示匿名身份'}
                  </div>
                </div>
                {!user?.access_token && (
                  <Button
                    className="mt-3 w-100"
                    onClick={handlePreviewEntry}
                    disabled={isEntering}>
                    {isEntering ? '正在进入...' : '进入匿名社区'}
                  </Button>
                )}
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
          {user?.access_token && (
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
          {(user?.role_id === 2 || user?.role_id === 3) && (
            <NavLink to="/community/moderation" className="community-nav-link">
              管理
            </NavLink>
          )}
        </nav>

        {bootError ? (
          <div className="alert alert-danger">{bootError}</div>
        ) : (
          <Outlet />
        )}
      </div>
      <div className="container-xxl pb-4">
        <Footer />
      </div>
    </div>
  );
};

export default memo(Layout);
