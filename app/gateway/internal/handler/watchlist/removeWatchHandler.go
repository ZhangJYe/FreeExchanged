// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package watchlist

import (
	"net/http"

	"freeexchanged/app/gateway/internal/logic/watchlist"
	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Remove currency pair from watchlist
func RemoveWatchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RemoveWatchReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := watchlist.NewRemoveWatchLogic(r.Context(), svcCtx)
		resp, err := l.RemoveWatch(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
