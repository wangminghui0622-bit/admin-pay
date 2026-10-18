package ali

import (
	"context"

	"github.com/go-pay/gopay"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
)

// pcPayStrategy 支付宝 PC 网页支付
type pcPayStrategy struct {
	*Client
}

func (a *pcPayStrategy) UnifiedOrder(ctx context.Context, req model.OrderUnifiedReq) (*model.OrderResp, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", req.OutTradeNo).
		Set("subject", req.Subject).
		Set("body", req.Body).
		Set("total_amount", formatAmount(req.Price)).
		Set("product_code", "FAST_INSTANT_TRADE_PAY").
		Set("qr_pay_mode", "2").
		Set("time_expire", req.ExpireTime.Format("2006-01-02 15:04:05"))
	displayMode := req.DisplayMode
	if displayMode == "" {
		displayMode = model.Url
	}
	a.client.SetReturnUrl(req.ReturnUrl)
	a.client.SetNotifyUrl(req.NotifyUrl)
	payUrl, err := a.client.TradePagePay(ctx, bm)
	if err != nil {
		return nil, err
	}
	return model.WaitingOf(&displayMode, &payUrl, req.OutTradeNo, payUrl), nil
}
