// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package article

import (
	"net/http"

	"freeexchanged/app/gateway/internal/logic/article"
	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Publish new article
func PublishHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PublishArticleReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := article.NewPublishLogic(r.Context(), svcCtx)
		resp, err := l.Publish(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
