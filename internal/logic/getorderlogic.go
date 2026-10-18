package logic

import (
	"context"

	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderLogic {
	return &GetOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetOrderLogic) GetOrder(in *pay.IDReq) (*pay.OrderInfo, error) {
	q := l.svcCtx.Query
	order, err := q.PayOrder.WithContext(l.ctx).Where(q.PayOrder.ID.Eq(int64(in.Id))).First()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	return toOrderInfo(order), nil
}
