package logic

import (
	"context"
	"strconv"
	"time"

	"github.com/wangminghui0622-bit/admin-common/dal/model"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateDemoRefundPaidLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateDemoRefundPaidLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDemoRefundPaidLogic {
	return &UpdateDemoRefundPaidLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateDemoRefundPaidLogic) UpdateDemoRefundPaid(in *pay.UpdateDemoRefundPaidReq) (*pay.BaseResp, error) {
	q := l.svcCtx.Query
	demoOrder, err := q.PayDemoOrder.WithContext(l.ctx).Where(q.PayDemoOrder.ID.Eq(int64(in.Id))).First()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	// 原版 bug：这里误比较了 PayOrderId，应为 PayRefundId
	if demoOrder.PayRefundID == nil || uint64(*demoOrder.PayRefundID) != in.PayRefundId {
		l.Errorf("[updateDemoRefundPaid][order(%d) 退款单不匹配(%d)]", in.Id, in.PayRefundId)
		return nil, errorx.New(errorx.InvalidArgument, "demo order refund fail refund order id error")
	}
	refund, err := q.PayRefund.WithContext(l.ctx).Where(q.PayRefund.ID.Eq(int64(in.PayRefundId))).First()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	if statusValue(refund.Status) != payStatusSuccess {
		return nil, errorx.New(errorx.InvalidArgument, "demo order refund fail refund not success")
	}
	if refund.RefundPrice != demoOrder.Price {
		l.Errorf("[updateDemoRefundPaid][order(%d) 退款金额不匹配]", demoOrder.ID)
		return nil, errorx.New(errorx.InvalidArgument, "demo order refund fail refund price not match")
	}
	if refund.MerchantRefundID != strconv.FormatInt(demoOrder.ID, 10)+"-refund" {
		l.Errorf("[updateDemoRefundPaid][order(%d) 退款单号不匹配(%s)]", demoOrder.ID, refund.MerchantRefundID)
		return nil, errorx.New(errorx.InvalidArgument, "demo order refund fail refund id not match")
	}

	refundTime := time.Now()
	if refund.SuccessTime != nil && !refund.SuccessTime.IsZero() {
		refundTime = *refund.SuccessTime
	}
	result := l.svcCtx.DB.WithContext(l.ctx).Model(&model.PayDemoOrder{}).
		Where("id = ?", demoOrder.ID).
		Updates(map[string]any{"refund_time": refundTime})
	if result.Error != nil {
		return nil, dbError(l.Logger, result.Error, in)
	}
	return &pay.BaseResp{Msg: msgSuccess}, nil
}
