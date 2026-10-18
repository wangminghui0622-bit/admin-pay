package logic

import (
	"context"

	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRefundByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRefundByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRefundByIdLogic {
	return &GetRefundByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetRefundByIdLogic) GetRefundById(in *pay.IDReq) (*pay.RefundInfo, error) {
	q := l.svcCtx.Query
	refund, err := q.PayRefund.WithContext(l.ctx).Where(q.PayRefund.ID.Eq(int64(in.Id))).First()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	return toRefundInfo(refund), nil
}
