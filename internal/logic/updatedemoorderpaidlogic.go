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

type UpdateDemoOrderPaidLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateDemoOrderPaidLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDemoOrderPaidLogic {
	return &UpdateDemoOrderPaidLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateDemoOrderPaidLogic) UpdateDemoOrderPaid(in *pay.UpdateDemoOrderPaidReq) (*pay.BaseResp, error) {
	q := l.svcCtx.Query
	demoOrder, err := q.PayDemoOrder.WithContext(l.ctx).Where(q.PayDemoOrder.ID.Eq(int64(in.Id))).First()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	if demoOrder.PayStatus {
		l.Errorf("[updateDemoOrderPaid][order(%d) 不处于待支付状态]", demoOrder.ID)
		return nil, errorx.New(errorx.InvalidArgument, "DEMO_ORDER_UPDATE_PAID_STATUS_NOT_UNPAID")
	}
	if demoOrder.PayOrderID == nil || uint64(*demoOrder.PayOrderID) != in.PayOrderId {
		l.Errorf("[updateDemoOrderPaid][order(%d) 支付订单号不匹配]", demoOrder.ID)
		return nil, errorx.New(errorx.InvalidArgument, "DEMO_ORDER_UPDATE_PAID_PAY_ORDER_ID_NOT_MATCH")
	}
	payOrder, err := q.PayOrder.WithContext(l.ctx).Where(q.PayOrder.ID.Eq(int64(in.PayOrderId))).First()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	if statusValue(payOrder.Status) != payStatusSuccess {
		l.Errorf("[updateDemoOrderPaid][order(%d) 支付订单未支付]", demoOrder.ID)
		return nil, errorx.New(errorx.InvalidArgument, "DEMO_ORDER_UPDATE_PAID_PAY_ORDER_PAID")
	}
	if payOrder.Price != demoOrder.Price {
		l.Errorf("[updateDemoOrderPaid][order(%d) 支付订单金额不匹配]", demoOrder.ID)
		return nil, errorx.New(errorx.InvalidArgument, "DEMO_ORDER_UPDATE_PAID_PAY_ORDER_PRICE_NOT_MATCH")
	}
	if payOrder.MerchantOrderID != strconv.FormatInt(demoOrder.ID, 10) {
		l.Errorf("[updateDemoOrderPaid][order(%d) 支付订单商户订单号不匹配]", demoOrder.ID)
		return nil, errorx.New(errorx.InvalidArgument, "DEMO_ORDER_UPDATE_PAID_PAY_ORDER_MERCHANT_ORDER_ID_NOT_MATCH")
	}

	updates := map[string]any{
		"pay_status": true,
		"pay_time":   time.Now(),
	}
	if payOrder.ChannelCode != nil {
		updates["pay_channel_code"] = *payOrder.ChannelCode
	}
	result := l.svcCtx.DB.WithContext(l.ctx).Model(&model.PayDemoOrder{}).
		Where("id = ? AND pay_status = ?", demoOrder.ID, false).Updates(updates)
	if result.Error != nil {
		return nil, dbError(l.Logger, result.Error, in)
	}
	if result.RowsAffected == 0 {
		return nil, errorx.New(errorx.InvalidArgument, "DEMO_ORDER_UPDATE_PAID_STATUS_NOT_UNPAID")
	}
	return &pay.BaseResp{Msg: msgSuccess}, nil
}
