// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"net/http"

	"freeexchanged/app/gateway/internal/logic/user"
	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 健康检查 (需要鉴权)
func PingHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PingReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := user.NewPingLogic(r.Context(), svcCtx)
		resp, err := l.Ping(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
