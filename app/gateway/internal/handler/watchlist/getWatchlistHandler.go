// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package watchlist

import (
	"net/http"

	"freeexchanged/app/gateway/internal/logic/watchlist"
	"freeexchanged/app/gateway/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Get watchlist with real-time rates
func GetWatchlistHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := watchlist.NewGetWatchlistLogic(r.Context(), svcCtx)
		resp, err := l.GetWatchlist()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
