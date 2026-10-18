package logic

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/hibiken/asynq"
	"github.com/wangminghui0622-bit/admin-common/dal/model"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const msgSuccess = "success"

// 与 proto PayStatus 一致
const (
	payStatusWaiting = int32(pay.PayStatus_PAY_WAITING)
	payStatusSuccess = int32(pay.PayStatus_PAY_SUCCESS)
	payStatusRefund  = int32(pay.PayStatus_PAY_REFUND)
	payStatusClosed  = int32(pay.PayStatus_PAY_CLOSED)
)

// dbError 将数据库错误映射为统一的业务错误，避免把底层错误细节暴露给调用方。
func dbError(logger logx.Logger, err error, detail any) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errorx.New(errorx.InvalidArgument, "target not found")
	}
	logger.Errorf("database error: %v, detail: %+v", err, detail)
	return errorx.New(errorx.ServerError, "database error")
}

// enqueueTask 入队 asynq 任务；未启用 asynq 时跳过，避免消费者缺失导致业务失败。
func enqueueTask(svcCtx *svc.ServiceContext, taskType string, payload any) error {
	if svcCtx.AsynqClient == nil {
		logx.Infof("asynq is disabled, skip task %s", taskType)
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		logx.Errorf("marshal task %s payload failed: %v", taskType, err)
		return errorx.New(errorx.ServerError, "enqueue task failed")
	}
	if _, err = svcCtx.AsynqClient.Enqueue(asynq.NewTask(taskType, body)); err != nil {
		logx.Errorf("enqueue task %s failed: %v", taskType, err)
		return errorx.New(errorx.ServerError, "enqueue task failed")
	}
	return nil
}

// normalizePage 返回合法的页码与每页数量。
func normalizePage(page, pageSize uint64) (int, int) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 10
	}
	return int(page), int(pageSize)
}

func i32Ptr(v int32) *int32 {
	return &v
}

func u64Ptr(v int64) *uint64 {
	n := uint64(v)
	return &n
}

func u64PtrFromInt64Ptr(v *int64) *uint64 {
	if v == nil {
		return nil
	}
	return u64Ptr(*v)
}

func f64PtrFromFloat32Ptr(v *float32) *float64 {
	if v == nil {
		return nil
	}
	n := float64(*v)
	return &n
}

func u32Ptr(v *int32) *uint32 {
	if v == nil {
		return nil
	}
	n := uint32(*v)
	return &n
}

func statusValue(v *int32) int32 {
	if v == nil {
		return payStatusWaiting
	}
	return *v
}

func strValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// timeMillis 输出 Unix 毫秒；零值时间输出 nil，避免给前端无意义的时间戳。
func timeMillis(t time.Time) *int64 {
	if t.IsZero() {
		return nil
	}
	v := t.UnixMilli()
	return &v
}

func timePtrMillis(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	return timeMillis(*t)
}

func marshalJSON(v any) string {
	body, err := json.Marshal(v)
	if err != nil {
		logx.Errorf("marshal json failed: %v", err)
		return ""
	}
	return string(body)
}

func extrasToJSON(extras map[string]string) *string {
	if len(extras) == 0 {
		return nil
	}
	body := marshalJSON(extras)
	return &body
}

func extrasFromJSON(data *string) map[string]string {
	extras := make(map[string]string)
	if data == nil || *data == "" {
		return extras
	}
	if err := json.Unmarshal([]byte(*data), &extras); err != nil {
		logx.Errorf("unmarshal channel extras failed: %v", err)
		return map[string]string{}
	}
	if extras == nil {
		extras = make(map[string]string)
	}
	return extras
}

func toOrderInfo(o *model.PayOrder) *pay.OrderInfo {
	if o == nil {
		return nil
	}
	return &pay.OrderInfo{
		Id:              u64Ptr(o.ID),
		CreatedAt:       timeMillis(o.CreatedAt),
		UpdatedAt:       timeMillis(o.UpdatedAt),
		Status:          u32Ptr(o.Status),
		ChannelCode:     o.ChannelCode,
		MerchantOrderId: &o.MerchantOrderID,
		Subject:         &o.Subject,
		Body:            &o.Body,
		Price:           &o.Price,
		ChannelFeeRate:  f64PtrFromFloat32Ptr(o.ChannelFeeRate),
		ChannelFeePrice: o.ChannelFeePrice,
		UserIp:          &o.UserIP,
		ExpireTime:      timeMillis(o.ExpireTime),
		SuccessTime:     timePtrMillis(o.SuccessTime),
		NotifyTime:      timePtrMillis(o.NotifyTime),
		ExtensionId:     u64PtrFromInt64Ptr(o.ExtensionID),
		No:              o.No,
		RefundPrice:     &o.RefundPrice,
		ChannelUserId:   o.ChannelUserID,
		ChannelOrderNo:  o.ChannelOrderNo,
	}
}

func toOrderExtensionInfo(e *model.PayOrderExtension) *pay.OrderExtensionInfo {
	if e == nil {
		return nil
	}
	return &pay.OrderExtensionInfo{
		Id:                u64Ptr(e.ID),
		CreatedAt:         timePtrMillis(e.CreatedAt),
		UpdatedAt:         timePtrMillis(e.UpdatedAt),
		Status:            u32Ptr(e.Status),
		No:                &e.No,
		OrderId:           u64Ptr(e.OrderID),
		ChannelCode:       &e.ChannelCode,
		UserIp:            &e.UserIP,
		ChannelExtras:     extrasFromJSON(e.ChannelExtras),
		ChannelErrorCode:  e.ChannelErrorCode,
		ChannelErrorMsg:   e.ChannelErrorMsg,
		ChannelNotifyData: e.ChannelNotifyData,
	}
}

func toRefundInfo(r *model.PayRefund) *pay.RefundInfo {
	if r == nil {
		return nil
	}
	return &pay.RefundInfo{
		Id:                u64Ptr(r.ID),
		CreatedAt:         timePtrMillis(r.CreatedAt),
		UpdatedAt:         timePtrMillis(r.UpdatedAt),
		Status:            u32Ptr(r.Status),
		No:                &r.No,
		ChannelCode:       &r.ChannelCode,
		OrderId:           u64Ptr(r.OrderID),
		OrderNo:           &r.OrderNo,
		MerchantOrderId:   &r.MerchantOrderID,
		MerchantRefundId:  &r.MerchantRefundID,
		PayPrice:          &r.PayPrice,
		RefundPrice:       &r.RefundPrice,
		Reason:            &r.Reason,
		UserIp:            r.UserIP,
		ChannelOrderNo:    &r.ChannelOrderNo,
		ChannelRefundNo:   r.ChannelRefundNo,
		SuccessTime:       timePtrMillis(r.SuccessTime),
		ChannelErrorCode:  r.ChannelErrorCode,
		ChannelErrorMsg:   r.ChannelErrorMsg,
		ChannelNotifyData: r.ChannelNotifyData,
	}
}

func toDemoOrderInfo(d *model.PayDemoOrder) *pay.DemoOrderInfo {
	if d == nil {
		return nil
	}
	return &pay.DemoOrderInfo{
		Id:             u64Ptr(d.ID),
		CreatedAt:      timePtrMillis(d.CreatedAt),
		UpdatedAt:      timePtrMillis(d.UpdatedAt),
		UserId:         &d.UserID,
		SpuId:          u64Ptr(d.SpuID),
		SpuName:        &d.SpuName,
		Price:          &d.Price,
		PayStatus:      &d.PayStatus,
		PayOrderId:     u64PtrFromInt64Ptr(d.PayOrderID),
		PayTime:        timePtrMillis(d.PayTime),
		PayChannelCode: d.PayChannelCode,
		PayRefundId:    u64PtrFromInt64Ptr(d.PayRefundID),
		RefundPrice:    d.RefundPrice,
		RefundTime:     timePtrMillis(d.RefundTime),
	}
}
