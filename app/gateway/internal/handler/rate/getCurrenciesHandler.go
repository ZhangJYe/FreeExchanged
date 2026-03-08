// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package rate

import (
	"net/http"

	ratelogic "freeexchanged/app/gateway/internal/logic/rate"
	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 获取支持的货币列表
func GetCurrenciesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetCurrenciesReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := ratelogic.NewGetCurrenciesLogic(r.Context(), svcCtx)
		resp, err := l.GetCurrencies(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
