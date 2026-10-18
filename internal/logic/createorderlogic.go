package logic

import (
	"context"
	"errors"
	"time"

	"github.com/wangminghui0622-bit/admin-common/dal/model"
	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type CreateOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrderLogic {
	return &CreateOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateOrder 创建支付订单；merchant_order_id 幂等，重复创建返回原订单编号。
func (l *CreateOrderLogic) CreateOrder(in *pay.OrderCreateReq) (*pay.BaseIDResp, error) {
	q := l.svcCtx.Query
	existing, err := q.PayOrder.WithContext(l.ctx).
		Where(q.PayOrder.MerchantOrderID.Eq(in.MerchantOrderId)).First()
	if err == nil {
		l.Infof("[createOrder][merchantOrderId(%s) 已经存在对应的支付单(%d)]", in.MerchantOrderId, existing.ID)
		return &pay.BaseIDResp{Id: uint64(existing.ID), Msg: "order already exists"}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, dbError(l.Logger, err, in.MerchantOrderId)
	}

	order := &model.PayOrder{
		Status:          i32Ptr(payStatusWaiting),
		MerchantOrderID: in.MerchantOrderId,
		Subject:         in.Subject,
		Body:            in.Body,
		Price:           in.Price,
		UserIP:          in.UserIp,
		ExpireTime:      time.Unix(in.ExpireTime, 0),
		RefundPrice:     0,
	}
	if err = q.PayOrder.WithContext(l.ctx).Create(order); err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	return &pay.BaseIDResp{Id: uint64(order.ID), Msg: msgSuccess}, nil
}
