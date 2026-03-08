// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ranking

import (
	"net/http"

	"freeexchanged/app/gateway/internal/logic/ranking"
	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取热榜
func GetTopHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetTopReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := ranking.NewGetTopLogic(r.Context(), svcCtx)
		resp, err := l.GetTop(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
