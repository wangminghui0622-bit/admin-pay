package logic

import (
	"context"

	"github.com/wangminghui0622-bit/admin-common/dal/model"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"
	"github.com/wangminghui0622-bit/admin-pay/payment"
	pmodel "github.com/wangminghui0622-bit/admin-pay/payment/model"
	"github.com/wangminghui0622-bit/admin-pay/utils/payno"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateRefundLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateRefundLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateRefundLogic {
	return &CreateRefundLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateRefundLogic) CreateRefund(in *pay.RefundCreateReq) (*pay.BaseIDResp, error) {
	order, err := l.validatePayOrderCanRefund(in)
	if err != nil {
		return nil, err
	}

	q := l.svcCtx.Query
	count, err := q.PayRefund.WithContext(l.ctx).
		Where(q.PayRefund.MerchantRefundID.Eq(in.MerchantRefundId)).Count()
	if err != nil {
		return nil, dbError(l.Logger, err, in.MerchantRefundId)
	}
	if count > 0 {
		return nil, errorx.New(errorx.InvalidArgument, "refund exists")
	}

	channelCode := strValue(order.ChannelCode)
	client, err := payment.GetClient(l.svcCtx.PayClient, channelCode)
	if err != nil {
		return nil, err
	}
	refundNo, err := payno.Generate(l.svcCtx.Redis, l.svcCtx.Config.PayProperties.RefundNoPrefix)
	if err != nil {
		l.Errorf("generate refund no failed: %v", err)
		return nil, errorx.New(errorx.ServerError, "generate refund no failed")
	}
	refund := &model.PayRefund{
		Status:           i32Ptr(payStatusWaiting),
		No:               refundNo,
		ChannelCode:      channelCode,
		OrderID:          order.ID,
		OrderNo:          strValue(order.No),
		MerchantOrderID:  in.MerchantOrderId,
		MerchantRefundID: in.MerchantRefundId,
		PayPrice:         order.Price,
		RefundPrice:      in.Price,
		Reason:           in.Reason,
		UserIP:           &in.UserIp,
		ChannelOrderNo:   strValue(order.ChannelOrderNo),
	}
	if err = q.PayRefund.WithContext(l.ctx).Create(refund); err != nil {
		return nil, dbError(l.Logger, err, in)
	}

	refundID := refund.ID
	refundNo = refund.No
	reason := refund.Reason
	payPrice := refund.PayPrice
	refundPrice := refund.RefundPrice
	orderNo := refund.OrderNo
	notifyUrl := l.svcCtx.Config.PayProperties.RefundNotifyUrl + "/" + channelCode
	go func() {
		newCtx := context.Background()
		refundResp, err := client.UnifiedRefund(newCtx, pmodel.RefundUnifiedReq{
			OutTradeNo:  orderNo,
			OutRefundNo: refundNo,
			Reason:      reason,
			PayPrice:    payPrice,
			RefundPrice: refundPrice,
			NotifyUrl:   notifyUrl,
		})
		if err != nil {
			logx.Errorf("[createRefund][refund(%d) 发起退款失败: %v]", refundID, err)
			return
		}
		if refundResp == nil {
			logx.Errorf("[createRefund][refund(%d) 渠道返回为空]", refundID)
			return
		}
		if err = NewNotifyRefundLogic(newCtx, l.svcCtx).ProcessRefundStatus(refundResp); err != nil {
			logx.Errorf("[createRefund][refund(%d) 处理退款结果失败: %v]", refundID, err)
		}
	}()

	return &pay.BaseIDResp{Id: uint64(refund.ID), Msg: msgSuccess}, nil
}

func (l *CreateRefundLogic) validatePayOrderCanRefund(in *pay.RefundCreateReq) (*model.PayOrder, error) {
	q := l.svcCtx.Query
	order, err := q.PayOrder.WithContext(l.ctx).
		Where(q.PayOrder.MerchantOrderID.Eq(in.MerchantOrderId)).First()
	if err != nil {
		return nil, dbError(l.Logger, err, in.MerchantOrderId)
	}
	status := statusValue(order.Status)
	if status != payStatusSuccess && status != payStatusRefund {
		return nil, errorx.New(errorx.InvalidArgument, "pay order refund fail status error")
	}
	if in.Price+order.RefundPrice > order.Price {
		return nil, errorx.New(errorx.InvalidArgument, "refund price exceed")
	}
	count, err := q.PayRefund.WithContext(l.ctx).
		Where(q.PayRefund.OrderID.Eq(order.ID), q.PayRefund.Status.Eq(payStatusWaiting)).Count()
	if err != nil {
		return nil, dbError(l.Logger, err, in)
	}
	if count > 0 {
		return nil, errorx.New(errorx.InvalidArgument, "refund has refunding")
	}
	return order, nil
}
