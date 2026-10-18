package logic

import (
	"context"
	"fmt"
	"time"

	"github.com/wangminghui0622-bit/admin-common/dal/model"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"
	"github.com/wangminghui0622-bit/admin-pay/payment"
	pmodel "github.com/wangminghui0622-bit/admin-pay/payment/model"
	"github.com/wangminghui0622-bit/admin-pay/utils/payno"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitPayOrderLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSubmitPayOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitPayOrderLogic {
	return &SubmitPayOrderLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SubmitPayOrderLogic) SubmitPayOrder(in *pay.OrderSubmitReq) (*pay.OrderSubmitResp, error) {
	order, err := l.validateOrderCanSubmit(in.Id)
	if err != nil {
		return nil, err
	}
	// 先确认渠道已启用，避免创建无意义的拓展单（同时修复原版 nil panic）
	client, err := payment.GetClient(l.svcCtx.PayClient, in.ChannelCode)
	if err != nil {
		return nil, err
	}
	no, err := payno.Generate(l.svcCtx.Redis, l.svcCtx.Config.PayProperties.OrderNoPrefix)
	if err != nil {
		l.Errorf("generate pay order no failed: %v", err)
		return nil, errorx.New(errorx.ServerError, "generate pay no failed")
	}
	if _, err = NewCreateOrderExtensionLogic(l.ctx, l.svcCtx).CreateOrderExtension(&pay.OrderCreateExtensionReq{
		OrderId:       uint64(order.ID),
		ChannelCode:   in.ChannelCode,
		ChannelExtras: in.ChannelExtras,
		No:            no,
		UserIp:        in.UserIp,
	}); err != nil {
		return nil, err
	}

	unifiedOrderResp, err := client.UnifiedOrder(l.ctx, in.ChannelCode, pmodel.OrderUnifiedReq{
		DisplayMode:   in.DisplayMode,
		UserIp:        in.UserIp,
		OutTradeNo:    no,
		Subject:       order.Subject,
		Body:          order.Body,
		NotifyUrl:     l.svcCtx.Config.PayProperties.OrderNotifyUrl + "/" + in.ChannelCode,
		ReturnUrl:     in.ReturnUrl,
		Price:         order.Price,
		ExpireTime:    order.ExpireTime,
		ChannelExtras: in.ChannelExtras,
	})
	if err != nil {
		return nil, err
	}
	if unifiedOrderResp == nil {
		l.Errorf("[submitPayOrder][order(%d) 渠道返回为空]", order.ID)
		return nil, errorx.New(errorx.ServerError, "pay channel response is empty")
	}
	// 修复原版对 string 指针使用 %d 格式化的问题
	if unifiedOrderResp.ChannelErrorCode != nil {
		return nil, errorx.New(errorx.InvalidArgument, fmt.Sprintf("pay channel error, code: %s, msg: %s",
			*unifiedOrderResp.ChannelErrorCode, strValue(unifiedOrderResp.ChannelErrorMsg)))
	}

	channelCode := in.ChannelCode
	resp := unifiedOrderResp
	go func() {
		if err := NewNotifyOrderLogic(context.Background(), l.svcCtx).NotifyOrder0(channelCode, resp); err != nil {
			logx.Errorf("[submitPayOrder][order(%d) 处理渠道同步结果失败: %v]", order.ID, err)
		}
	}()

	return &pay.OrderSubmitResp{
		Status:         uint32(statusValue(order.Status)),
		DisplayMode:    unifiedOrderResp.DisplayMode,
		DisplayContent: unifiedOrderResp.DisplayContent,
	}, nil
}

func (l *SubmitPayOrderLogic) validateOrderCanSubmit(id uint64) (*model.PayOrder, error) {
	q := l.svcCtx.Query
	order, err := q.PayOrder.WithContext(l.ctx).Where(q.PayOrder.ID.Eq(int64(id))).First()
	if err != nil {
		return nil, dbError(l.Logger, err, id)
	}
	switch statusValue(order.Status) {
	case payStatusSuccess:
		return nil, errorx.New(errorx.InvalidArgument, "pay order status is success")
	case payStatusWaiting:
	default:
		return nil, errorx.New(errorx.InvalidArgument, "pay order status is not waiting")
	}
	if time.Now().After(order.ExpireTime) {
		return nil, errorx.New(errorx.InvalidArgument, "pay order is expired")
	}
	return order, nil
}
