package logic

import (
	"context"
	"time"

	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrderPageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOrderPageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderPageLogic {
	return &GetOrderPageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetOrderPageLogic) GetOrderPage(in *pay.OrderPageReq) (*pay.OrderListResp, error) {
	page, pageSize := normalizePage(in.Page, in.PageSize)
	q := l.svcCtx.Query
	dao := q.PayOrder.WithContext(l.ctx)
	if in.ChannelCode != nil {
		dao = dao.Where(q.PayOrder.ChannelCode.Eq(*in.ChannelCode))
	}
	// 原版 bug：这里误用了 ChannelCode，必须按 merchant_order_id 过滤
	if in.MerchantOrderId != nil {
		dao = dao.Where(q.PayOrder.MerchantOrderID.Like("%" + *in.MerchantOrderId + "%"))
	}
	if in.ChannelOrderNo != nil {
		dao = dao.Where(q.PayOrder.ChannelOrderNo.Like("%" + *in.ChannelOrderNo + "%"))
	}
	if in.No != nil {
		dao = dao.Where(q.PayOrder.No.Like("%" + *in.No + "%"))
	}
	if in.Status != nil {
		dao = dao.Where(q.PayOrder.Status.Eq(int32(*in.Status)))
	}
	// 原版 bug：未判断 create_time 长度导致下标越界 panic
	if len(in.CreateTime) > 0 && in.CreateTime[0] > 0 {
		dao = dao.Where(q.PayOrder.CreatedAt.Gte(time.Unix(in.CreateTime[0], 0)))
	}
	if len(in.CreateTime) > 1 && in.CreateTime[1] > 0 {
		dao = dao.Where(q.PayOrder.CreatedAt.Lte(time.Unix(in.CreateTime[1], 0)))
	}

	total, err := dao.Count()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	orders, err := dao.Order(q.PayOrder.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	infos := make([]*pay.OrderInfo, 0, len(orders))
	for _, order := range orders {
		infos = append(infos, toOrderInfo(order))
	}
	return &pay.OrderListResp{Total: uint64(total), Data: infos}, nil
}
