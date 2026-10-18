package logic

import (
	"context"

	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrderExtensionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOrderExtensionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderExtensionLogic {
	return &GetOrderExtensionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetOrderExtensionLogic) GetOrderExtension(in *pay.IDReq) (*pay.OrderExtensionInfo, error) {
	q := l.svcCtx.Query
	extension, err := q.PayOrderExtension.WithContext(l.ctx).Where(q.PayOrderExtension.ID.Eq(int64(in.Id))).First()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	return toOrderExtensionInfo(extension), nil
}
