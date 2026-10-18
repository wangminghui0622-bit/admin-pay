package ali

import (
	"context"

	"github.com/go-pay/gopay"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
)

// wapPayStrategy 支付宝手机网站支付
type wapPayStrategy struct {
	*Client
}

func (a *wapPayStrategy) UnifiedOrder(ctx context.Context, req model.OrderUnifiedReq) (*model.OrderResp, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", req.OutTradeNo).
		Set("subject", req.Subject).
		Set("body", req.Body).
		Set("total_amount", formatAmount(req.Price)).
		Set("product_code", "QUICK_WAP_WAY").
		Set("quit_url", req.ReturnUrl)
	a.client.SetReturnUrl(req.ReturnUrl)
	a.client.SetNotifyUrl(req.NotifyUrl)
	payUrl, err := a.client.TradeWapPay(ctx, bm)
	if err != nil {
		return nil, err
	}
	return model.WaitingOf(model.Ptr(model.Url), &payUrl, req.OutTradeNo, payUrl), nil
}
