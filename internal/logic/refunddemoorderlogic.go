package logic

import (
	"context"
	"fmt"
	"strconv"

	"github.com/wangminghui0622-bit/admin-common/dal/model"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefundDemoOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRefundDemoOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefundDemoOrderLogic {
	return &RefundDemoOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RefundDemoOrderLogic) RefundDemoOrder(in *pay.RefundDemoOrderReq) (*pay.BaseResp, error) {
	q := l.svcCtx.Query
	demoOrder, err := q.PayDemoOrder.WithContext(l.ctx).Where(q.PayDemoOrder.ID.Eq(int64(in.Id))).First()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	if !demoOrder.PayStatus {
		return nil, errorx.New(errorx.InvalidArgument, "demo order not pay")
	}
	if demoOrder.PayRefundID != nil {
		return nil, errorx.New(errorx.InvalidArgument, "demo order already refund")
	}

	refundResp, err := NewCreateRefundLogic(l.ctx, l.svcCtx).CreateRefund(&pay.RefundCreateReq{
		UserIp:           in.UserIp,
		MerchantOrderId:  strconv.FormatInt(demoOrder.ID, 10),
		MerchantRefundId: fmt.Sprintf("%d-refund", demoOrder.ID),
		Reason:           "想退钱",
		Price:            demoOrder.Price,
	})
	if err != nil {
		return nil, err
	}

	refundPrice := demoOrder.Price
	result := l.svcCtx.DB.WithContext(l.ctx).Model(&model.PayDemoOrder{}).
		Where("id = ?", demoOrder.ID).
		Updates(map[string]any{
			"pay_refund_id": int64(refundResp.Id),
			"refund_price":  refundPrice,
		})
	if result.Error != nil {
		return nil, dbError(l.Logger, result.Error, in)
	}
	return &pay.BaseResp{Msg: msgSuccess}, nil
}
