package weixin

import (
	"context"

	"github.com/go-pay/gopay/wechat/v3"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
)

// nativePayStrategy 微信 Native 扫码支付
type nativePayStrategy struct {
	*Client
}

func (w *nativePayStrategy) UnifiedOrder(ctx context.Context, req model.OrderUnifiedReq) (*model.OrderResp, error) {
	bm := w.buildPayUnifiedOrderBm(req)
	wxRsp, err := w.client.V3TransactionNative(ctx, bm)
	if err != nil {
		return nil, err
	}
	if wxRsp.Code != wechat.Success {
		return nil, errorx.New(errorx.InvalidArgument, wxRsp.Error)
	}
	return model.WaitingOf(model.Ptr(model.QrCode), model.Ptr(wxRsp.Response.CodeUrl), req.OutTradeNo, wxRsp.Response), nil
}
