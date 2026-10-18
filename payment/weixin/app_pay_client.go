package weixin

import (
	"context"
	"encoding/json"

	"github.com/go-pay/gopay/wechat/v3"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
)

// appPayStrategy 微信 App 支付
type appPayStrategy struct {
	*Client
}

func (w *appPayStrategy) UnifiedOrder(ctx context.Context, req model.OrderUnifiedReq) (*model.OrderResp, error) {
	bm := w.buildPayUnifiedOrderBm(req)
	wxRsp, err := w.client.V3TransactionApp(ctx, bm)
	if err != nil {
		return nil, err
	}
	if wxRsp.Code != wechat.Success {
		return nil, errorx.New(errorx.InvalidArgument, wxRsp.Error)
	}
	result, err := w.client.PaySignOfApp(w.Config.AppId, wxRsp.Response.PrepayId)
	if err != nil {
		return nil, err
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return model.WaitingOf(model.Ptr(model.App), model.Ptr(string(jsonData)), req.OutTradeNo, result), nil
}
