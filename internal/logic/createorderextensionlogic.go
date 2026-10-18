package logic

import (
	"context"

	"github.com/wangminghui0622-bit/admin-common/dal/model"
	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateOrderExtensionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateOrderExtensionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrderExtensionLogic {
	return &CreateOrderExtensionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateOrderExtensionLogic) CreateOrderExtension(in *pay.OrderCreateExtensionReq) (*pay.BaseIDResp, error) {
	extension := &model.PayOrderExtension{
		Status:        i32Ptr(payStatusWaiting), // 状态强制为待支付，忽略入参
		No:            in.No,
		OrderID:       int64(in.OrderId),
		ChannelCode:   in.ChannelCode,
		UserIP:        in.UserIp,
		ChannelExtras: extrasToJSON(in.ChannelExtras),
	}
	if err := l.svcCtx.Query.PayOrderExtension.WithContext(l.ctx).Create(extension); err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	return &pay.BaseIDResp{Id: uint64(extension.ID), Msg: msgSuccess}, nil
}
