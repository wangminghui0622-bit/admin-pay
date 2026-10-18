package logic

import (
	"context"

	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetListDemoOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetListDemoOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetListDemoOrderLogic {
	return &GetListDemoOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetListDemoOrderLogic) GetListDemoOrder(in *pay.DemoOrderPageReq) (*pay.DemoOrderListResp, error) {
	page, pageSize := normalizePage(in.Page, in.PageSize)
	q := l.svcCtx.Query
	dao := q.PayDemoOrder.WithContext(l.ctx)
	total, err := dao.Count()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	demoOrders, err := dao.Order(q.PayDemoOrder.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	infos := make([]*pay.DemoOrderInfo, 0, len(demoOrders))
	for _, demoOrder := range demoOrders {
		infos = append(infos, toDemoOrderInfo(demoOrder))
	}
	return &pay.DemoOrderListResp{Total: uint64(total), DemoOrderList: infos}, nil
}
