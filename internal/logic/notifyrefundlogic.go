package logic

import (
	"context"
	"time"

	"github.com/wangminghui0622-bit/admin-common/dal/model"
	"github.com/wangminghui0622-bit/admin-common/dal/query"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"
	"github.com/wangminghui0622-bit/admin-pay/payment"
	pmodel "github.com/wangminghui0622-bit/admin-pay/payment/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type NotifyRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewNotifyRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyRefundLogic {
	return &NotifyRefundLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *NotifyRefundLogic) NotifyRefund(in *pay.NotifyRefundReq) (*pay.BaseResp, error) {
	client, err := payment.GetClient(l.svcCtx.PayClient, in.ChannelCode)
	if err != nil {
		return nil, err
	}
	resp, err := client.ParseRefundNotify(in.R)
	if err != nil {
		l.Errorf("parse refund notify failed: %v", err)
		return nil, errorx.New(errorx.InvalidArgument, "parse refund notify error")
	}
	if err = l.ProcessRefundStatus(resp); err != nil {
		return nil, err
	}
	return &pay.BaseResp{Msg: msgSuccess}, nil
}

// ProcessRefundStatus 处理退款结果：成功则更新退款单与订单，失败保持等待（无轮询任务）。
func (l *NotifyRefundLogic) ProcessRefundStatus(notify *pmodel.RefundResp) error {
	if notify == nil {
		return errorx.New(errorx.ServerError, "pay channel response is empty")
	}
	switch notify.Status {
	case pmodel.SUCCESS:
		return l.notifyRefundSuccess(notify)
	default:
		// FAILURE/ERROR、WAITING 均不做处理
		return nil
	}
}

func (l *NotifyRefundLogic) notifyRefundSuccess(resp *pmodel.RefundResp) error {
	q := l.svcCtx.Query
	refund, err := q.PayRefund.WithContext(l.ctx).
		Where(q.PayRefund.No.Eq(resp.OutRefundNo)).First()
	if err != nil {
		return dbError(l.Logger, err, resp.OutRefundNo)
	}
	if statusValue(refund.Status) == payStatusSuccess {
		l.Infof("[notifyRefundSuccess][refund(%d) 已经是已退款，无需更新]", refund.ID)
		return nil // 修复原版重复通知返回错误的问题
	}
	if statusValue(refund.Status) != payStatusWaiting {
		return errorx.New(errorx.InvalidArgument, "refund status is not waiting")
	}

	successTime := resp.SuccessTime
	if successTime.IsZero() {
		successTime = time.Now()
	}
	notifyData := marshalJSON(resp.RawData)
	changed := false
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		txQuery := query.UsePay(tx)
		current, err := txQuery.PayRefund.WithContext(l.ctx).Where(txQuery.PayRefund.ID.Eq(refund.ID)).First()
		if err != nil {
			return dbError(l.Logger, err, resp.OutRefundNo)
		}
		if statusValue(current.Status) == payStatusSuccess {
			return nil // 并发下已被其它回调处理
		}
		if statusValue(current.Status) != payStatusWaiting {
			return errorx.New(errorx.InvalidArgument, "refund status is not waiting")
		}

		result := tx.WithContext(l.ctx).Model(&model.PayRefund{}).
			Where("id = ? AND status = ?", current.ID, payStatusWaiting).
			Updates(map[string]any{
				"status":              payStatusSuccess,
				"success_time":        successTime,
				"channel_refund_no":   resp.ChannelRefundNo,
				"channel_notify_data": notifyData,
			})
		if result.Error != nil {
			return dbError(l.Logger, result.Error, resp)
		}
		if result.RowsAffected == 0 {
			return errorx.New(errorx.InvalidArgument, "refund status is not waiting")
		}

		order, err := txQuery.PayOrder.WithContext(l.ctx).Where(txQuery.PayOrder.ID.Eq(current.OrderID)).First()
		if err != nil {
			return dbError(l.Logger, err, current.OrderID)
		}
		orderStatus := statusValue(order.Status)
		if orderStatus != payStatusSuccess && orderStatus != payStatusRefund {
			return errorx.New(errorx.InvalidArgument, "pay order refund is fail status error")
		}
		if order.RefundPrice+current.RefundPrice > order.Price {
			return errorx.New(errorx.InvalidArgument, "refund price is error")
		}
		result = tx.WithContext(l.ctx).Model(&model.PayOrder{}).
			Where("id = ? AND status IN ?", order.ID, []int32{payStatusSuccess, payStatusRefund}).
			Updates(map[string]any{
				"refund_price": order.RefundPrice + current.RefundPrice,
				"status":       payStatusRefund,
			})
		if result.Error != nil {
			return dbError(l.Logger, result.Error, order)
		}
		if result.RowsAffected == 0 {
			return errorx.New(errorx.InvalidArgument, "pay order refund is fail status error")
		}
		changed = true
		return nil
	})
	if err != nil {
		return err
	}
	if !changed {
		l.Infof("[notifyRefundSuccess][refund(%d) 已经被处理，无需重复入队]", refund.ID)
		return nil
	}
	l.Infof("[notifyRefundSuccess][refund(%d) 退款成功]", refund.ID)
	return enqueueTask(l.svcCtx, "pay_refund_success_notify", struct {
		MerchantOrderId string `json:"merchantOrderId"`
		PayRefundId     uint64 `json:"payRefundId"`
	}{
		MerchantOrderId: refund.MerchantOrderID,
		PayRefundId:     uint64(refund.ID),
	})
}
