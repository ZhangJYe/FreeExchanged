// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package interaction

import (
	"net/http"

	"freeexchanged/app/gateway/internal/logic/interaction"
	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 点赞/取消点赞
func LikeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LikeReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := interaction.NewLikeLogic(r.Context(), svcCtx)
		resp, err := l.Like(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
