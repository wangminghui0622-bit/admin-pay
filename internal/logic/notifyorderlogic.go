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

type NotifyOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewNotifyOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *NotifyOrderLogic {
	return &NotifyOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *NotifyOrderLogic) NotifyOrder(in *pay.NotifyOrderReq) (*pay.BaseResp, error) {
	client, err := payment.GetClient(l.svcCtx.PayClient, in.Code)
	if err != nil {
		return nil, err
	}
	resp, err := client.ParseOrderNotify(in.R)
	if err != nil {
		return nil, err
	}
	if err = l.NotifyOrder0(in.Code, resp); err != nil {
		return nil, err
	}
	return &pay.BaseResp{Msg: msgSuccess}, nil
}

// NotifyOrder0 处理渠道返回的支付结果，既用于异步回调，也用于支付下单后的同步结果。
func (l *NotifyOrderLogic) NotifyOrder0(channelCode string, resp *pmodel.OrderResp) error {
	if resp == nil {
		return errorx.New(errorx.ServerError, "pay channel response is empty")
	}
	switch resp.Status {
	case pmodel.SUCCESS:
		return l.notifyOrderSuccess(channelCode, resp)
	case pmodel.CLOSED:
		return l.notifyOrderClosed(resp)
	default:
		return nil
	}
}

// notifyOrderSuccess 支付成功：乐观锁更新拓展单与订单，然后入队业务通知。
func (l *NotifyOrderLogic) notifyOrderSuccess(channelCode string, resp *pmodel.OrderResp) error {
	changed := false
	var orderID int64
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		txQuery := query.UsePay(tx)
		extension, err := txQuery.PayOrderExtension.WithContext(l.ctx).
			Where(txQuery.PayOrderExtension.No.Eq(resp.OutTradeNo)).First()
		if err != nil {
			return dbError(l.Logger, err, resp.OutTradeNo)
		}
		switch statusValue(extension.Status) {
		case payStatusSuccess:
			// 已处理过，继续检查订单是否也已完成，保证部分失败后可重试补齐
		case payStatusWaiting:
			result := tx.WithContext(l.ctx).Model(&model.PayOrderExtension{}).
				Where("id = ? AND status = ?", extension.ID, payStatusWaiting).
				Updates(map[string]any{
					"status":              payStatusSuccess,
					"channel_notify_data": marshalJSON(resp),
				})
			if result.Error != nil {
				return dbError(l.Logger, result.Error, resp)
			}
			if result.RowsAffected == 0 {
				return errorx.New(errorx.InvalidArgument, "pay order extension status is not waiting")
			}
			changed = true
		default:
			return errorx.New(errorx.InvalidArgument, "pay order extension status is not waiting")
		}
		orderID = extension.OrderID

		order, err := txQuery.PayOrder.WithContext(l.ctx).Where(txQuery.PayOrder.ID.Eq(extension.OrderID)).First()
		if err != nil {
			return dbError(l.Logger, err, resp)
		}
		if statusValue(order.Status) == payStatusSuccess && order.ExtensionID != nil && *order.ExtensionID == extension.ID {
			return nil // 拓展单与订单均已处理，完全幂等
		}
		if statusValue(order.Status) != payStatusWaiting {
			return errorx.New(errorx.InvalidArgument, "pay order status is not waiting")
		}
		successTime := resp.SuccessTime
		if successTime.IsZero() {
			successTime = time.Now()
		}
		updates := map[string]any{
			"status":            payStatusSuccess,
			"channel_code":      channelCode,
			"success_time":      successTime,
			"notify_time":       time.Now(), // 修复原版 notify_time 从不写入的 bug
			"extension_id":      extension.ID,
			"no":                extension.No,
			"channel_order_no":  resp.ChannelOrderNo,
			"channel_fee_rate":  float64(0),
			"channel_fee_price": int32(0),
		}
		if resp.ChannelUserId != nil {
			updates["channel_user_id"] = *resp.ChannelUserId
		}
		result := tx.WithContext(l.ctx).Model(&model.PayOrder{}).
			Where("id = ? AND status = ?", order.ID, payStatusWaiting).
			Updates(updates)
		if result.Error != nil {
			return dbError(l.Logger, result.Error, resp)
		}
		if result.RowsAffected == 0 {
			return errorx.New(errorx.InvalidArgument, "pay order status is not waiting")
		}
		changed = true
		return nil
	})
	if err != nil {
		return err
	}
	if !changed {
		l.Infof("[notifyOrderSuccess][order(%d) 已经是已支付，无需更新]", orderID)
		return nil
	}

	q := l.svcCtx.Query
	order, err := q.PayOrder.WithContext(l.ctx).Where(q.PayOrder.ID.Eq(orderID)).First()
	if err != nil {
		return dbError(l.Logger, err, orderID)
	}
	return enqueueTask(l.svcCtx, "pay_order_success_notify", struct {
		MerchantOrderId string `json:"merchantOrderId"`
		PayOrderId      uint64 `json:"payOrderId"`
	}{
		MerchantOrderId: order.MerchantOrderID,
		PayOrderId:      uint64(order.ID),
	})
}

// notifyOrderClosed 支付关闭：记录渠道错误信息；重复通知保持幂等。
func (l *NotifyOrderLogic) notifyOrderClosed(resp *pmodel.OrderResp) error {
	q := l.svcCtx.Query
	extension, err := q.PayOrderExtension.WithContext(l.ctx).
		Where(q.PayOrderExtension.No.Eq(resp.OutTradeNo)).First()
	if err != nil {
		return dbError(l.Logger, err, resp.OutTradeNo)
	}
	switch statusValue(extension.Status) {
	case payStatusClosed:
		l.Infof("[notifyOrderClosed][orderExtension(%d) 已经是已关闭，无需更新]", extension.ID)
		return nil
	case payStatusSuccess:
		l.Infof("[notifyOrderClosed][orderExtension(%d) 已经是已支付，无需更新]", extension.ID)
		return nil
	case payStatusWaiting:
	default:
		return errorx.New(errorx.InvalidArgument, "pay order extension status is not waiting")
	}

	updates := map[string]any{
		"status":              payStatusClosed,
		"channel_notify_data": marshalJSON(resp),
	}
	if resp.ChannelErrorCode != nil {
		updates["channel_error_code"] = *resp.ChannelErrorCode
	}
	if resp.ChannelErrorMsg != nil {
		updates["channel_error_msg"] = *resp.ChannelErrorMsg
	}
	result := l.svcCtx.DB.WithContext(l.ctx).Model(&model.PayOrderExtension{}).
		Where("id = ? AND status = ?", extension.ID, payStatusWaiting).
		Updates(updates)
	if result.Error != nil {
		return dbError(l.Logger, result.Error, resp)
	}
	if result.RowsAffected == 0 {
		return errorx.New(errorx.InvalidArgument, "pay order extension status is not waiting")
	}
	l.Infof("[notifyOrderClosed][orderExtension(%d) 更新为已关闭]", extension.ID)
	return nil
}
