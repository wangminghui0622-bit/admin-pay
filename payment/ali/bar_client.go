package ali

import (
	"context"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
)

// barPayStrategy 支付宝条码支付
type barPayStrategy struct {
	*Client
}

func (a *barPayStrategy) UnifiedOrder(ctx context.Context, req model.OrderUnifiedReq) (*model.OrderResp, error) {
	authCode, ok := req.ChannelExtras["auth_code"]
	if !ok || authCode == "" {
		return nil, errorx.New(errorx.InvalidArgument, "auth_code is required")
	}
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", req.OutTradeNo).
		Set("subject", req.Subject).
		Set("body", req.Body).
		Set("total_amount", formatAmount(req.Price)).
		Set("scene", "bar_code").
		Set("auth_code", authCode)
	a.client.SetReturnUrl(req.ReturnUrl)
	a.client.SetNotifyUrl(req.NotifyUrl)
	resp, err := a.client.TradePay(ctx, bm)
	if err != nil {
		if bizErr, ok := alipay.IsBizError(err); ok {
			return model.CloseOf(bizErr.SubCode, bizErr.SubMsg, req.OutTradeNo, bizErr), nil
		}
		return nil, err
	}
	if resp.Response.Code == "10000" { // 免密支付，直接成功
		return model.SuccessOf(resp.Response.TradeNo, resp.Response.BuyerUserId, time.Now(), resp.Response.OutTradeNo, resp.Response), nil
	}
	// 10003 等状态需要用户输入密码，等待渠道异步通知
	return model.WaitingOf(model.Ptr(model.BarCode), model.Ptr(""), req.OutTradeNo, resp.Response), nil
}
