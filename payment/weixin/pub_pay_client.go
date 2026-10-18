package weixin

import (
	"context"
	"encoding/json"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
)

// pubPayStrategy 微信 JSAPI 支付（公众号 / 小程序）
type pubPayStrategy struct {
	*Client
}

func (w *pubPayStrategy) UnifiedOrder(ctx context.Context, req model.OrderUnifiedReq) (*model.OrderResp, error) {
	openid, err := getOpenid(req)
	if err != nil {
		return nil, err
	}
	bm := w.buildPayUnifiedOrderBm(req).SetBodyMap("payer", func(bm gopay.BodyMap) {
		bm.Set("openid", openid)
	})
	wxRsp, err := w.client.V3TransactionJsapi(ctx, bm)
	if err != nil {
		return nil, err
	}
	if wxRsp.Code != wechat.Success {
		return nil, errorx.New(errorx.InvalidArgument, wxRsp.Error)
	}
	result, err := w.client.PaySignOfJSAPI(w.Config.AppId, wxRsp.Response.PrepayId)
	if err != nil {
		return nil, err
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return model.WaitingOf(model.Ptr(model.App), model.Ptr(string(jsonData)), req.OutTradeNo, result), nil
}

func getOpenid(req model.OrderUnifiedReq) (string, error) {
	openid, ok := req.ChannelExtras["openid"]
	if !ok || openid == "" {
		return "", errorx.New(errorx.InvalidArgument, "openid is required")
	}
	return openid, nil
}
