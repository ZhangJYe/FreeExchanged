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

// 获取汇率（公开接口，无需鉴权）
func GetRateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetRateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := ratelogic.NewGetRateLogic(r.Context(), svcCtx)
		resp, err := l.GetRate(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
