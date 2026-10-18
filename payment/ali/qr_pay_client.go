package ali

import (
	"context"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
)

// qrPayStrategy 支付宝扫码支付
type qrPayStrategy struct {
	*Client
}

func (a *qrPayStrategy) UnifiedOrder(ctx context.Context, req model.OrderUnifiedReq) (*model.OrderResp, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", req.OutTradeNo).
		Set("subject", req.Subject).
		Set("body", req.Body).
		Set("total_amount", formatAmount(req.Price)).
		Set("product_code", "FACE_TO_FACE_PAYMENT")
	a.client.SetReturnUrl(req.ReturnUrl)
	a.client.SetNotifyUrl(req.NotifyUrl)
	resp, err := a.client.TradePrecreate(ctx, bm)
	if err != nil {
		if bizErr, ok := alipay.IsBizError(err); ok {
			return model.CloseOf(bizErr.SubCode, bizErr.SubMsg, req.OutTradeNo, bizErr), nil
		}
		return nil, err
	}
	return model.WaitingOf(model.Ptr(model.QrCode), &resp.Response.QrCode, req.OutTradeNo, resp.Response), nil
}
