package logic

import (
	"context"
	"time"

	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRefundPageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRefundPageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRefundPageLogic {
	return &GetRefundPageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetRefundPageLogic) GetRefundPage(in *pay.RefundPageReq) (*pay.RefundPageResp, error) {
	page, pageSize := normalizePage(in.Page, in.PageSize)
	q := l.svcCtx.Query
	dao := q.PayRefund.WithContext(l.ctx)
	if in.ChannelCode != nil {
		dao = dao.Where(q.PayRefund.ChannelCode.Eq(*in.ChannelCode))
	}
	if in.MerchantOrderId != nil {
		dao = dao.Where(q.PayRefund.MerchantOrderID.Like("%" + *in.MerchantOrderId + "%"))
	}
	if in.MerchantRefundId != nil {
		dao = dao.Where(q.PayRefund.MerchantRefundID.Like("%" + *in.MerchantRefundId + "%"))
	}
	if in.ChannelOrderNo != nil {
		dao = dao.Where(q.PayRefund.ChannelOrderNo.Like("%" + *in.ChannelOrderNo + "%"))
	}
	if in.ChannelRefundNo != nil {
		dao = dao.Where(q.PayRefund.ChannelRefundNo.Like("%" + *in.ChannelRefundNo + "%"))
	}
	if in.Status != nil {
		dao = dao.Where(q.PayRefund.Status.Eq(int32(*in.Status)))
	}
	// 修复原版未判断 create_time 长度导致的越界 panic
	if len(in.CreateTime) > 0 && in.CreateTime[0] > 0 {
		dao = dao.Where(q.PayRefund.CreatedAt.Gte(time.Unix(in.CreateTime[0], 0)))
	}
	if len(in.CreateTime) > 1 && in.CreateTime[1] > 0 {
		dao = dao.Where(q.PayRefund.CreatedAt.Lte(time.Unix(in.CreateTime[1], 0)))
	}

	total, err := dao.Count()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	refunds, err := dao.Order(q.PayRefund.ID.Desc()).Offset((page - 1) * pageSize).Limit(pageSize).Find()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	infos := make([]*pay.RefundInfo, 0, len(refunds))
	for _, refund := range refunds {
		infos = append(infos, toRefundInfo(refund))
	}
	return &pay.RefundPageResp{Total: uint64(total), Data: infos}, nil
}
