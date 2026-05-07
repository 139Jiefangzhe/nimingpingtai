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

package controller

import (
	"io"
	"net/http"
	"strings"

	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/service/wecom"
	"github.com/gin-gonic/gin"
)

type WeComController struct {
	wecomService *wecom.Service
}

func NewWeComController(wecomService *wecom.Service) *WeComController {
	return &WeComController{wecomService: wecomService}
}

func (wc *WeComController) AuthStart(ctx *gin.Context) {
	resp, err := wc.wecomService.GetAuthorizationURL(ctx.Query("return_to"))
	if err != nil {
		handler.HandleResponse(ctx, err, nil)
		return
	}
	if wantsWeComAuthStartJSON(ctx) {
		handler.HandleResponse(ctx, nil, resp)
		return
	}
	ctx.Redirect(http.StatusFound, resp.AuthorizationURL)
}

func wantsWeComAuthStartJSON(ctx *gin.Context) bool {
	if strings.EqualFold(ctx.GetHeader("X-Requested-With"), "XMLHttpRequest") {
		return true
	}

	accept := strings.ToLower(ctx.GetHeader("Accept"))
	if strings.Contains(accept, "application/json") &&
		!strings.Contains(accept, "text/html") {
		return true
	}

	if strings.EqualFold(ctx.GetHeader("Sec-Fetch-Dest"), "empty") {
		return true
	}

	return false
}

func (wc *WeComController) AuthCallback(ctx *gin.Context) {
	resp, err := wc.wecomService.HandleAuthCallback(ctx, ctx.Query("code"), ctx.Query("state"))
	if err != nil {
		handler.HandleResponse(ctx, err, nil)
		return
	}
	if resp.RedirectURL != "" {
		ctx.Redirect(http.StatusFound, resp.RedirectURL)
		return
	}
	handler.HandleResponse(ctx, nil, resp)
}

func (wc *WeComController) Callback(ctx *gin.Context) {
	if echostr := ctx.Query("echostr"); echostr != "" {
		resp, err := wc.wecomService.VerifyURL(
			ctx.Query("msg_signature"),
			ctx.Query("timestamp"),
			ctx.Query("nonce"),
			echostr,
		)
		if err != nil {
			handler.HandleResponse(ctx, err, nil)
			return
		}
		ctx.String(http.StatusOK, resp)
		return
	}
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		handler.HandleResponse(ctx, err, nil)
		return
	}
	err = wc.wecomService.HandleEventCallback(
		ctx,
		ctx.Query("msg_signature"),
		ctx.Query("timestamp"),
		ctx.Query("nonce"),
		body,
	)
	if err != nil {
		handler.HandleResponse(ctx, err, nil)
		return
	}
	ctx.String(http.StatusOK, "success")
}
