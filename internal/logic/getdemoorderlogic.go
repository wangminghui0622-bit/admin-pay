package logic

import (
	"context"

	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDemoOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDemoOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDemoOrderLogic {
	return &GetDemoOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDemoOrderLogic) GetDemoOrder(in *pay.IDReq) (*pay.DemoOrderInfo, error) {
	q := l.svcCtx.Query
	demoOrder, err := q.PayDemoOrder.WithContext(l.ctx).Where(q.PayDemoOrder.ID.Eq(int64(in.Id))).First()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	// 与原版 list 保持一致，统一返回 Unix 毫秒
	return toDemoOrderInfo(demoOrder), nil
}
