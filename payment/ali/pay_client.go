package ali

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// ClientConfig 支付宝证书模式配置，实现 model.ClientConfig 接口
type ClientConfig struct {
	AppId                   string `json:"appId"`
	SignType                string `json:"signType"`
	PrivateKey              string `json:"privateKey"`
	AppPublicContent        []byte `json:"appPublicContent"`
	AlipayPublicContentRSA2 []byte `json:"alipayPublicContentRSA2"`
	AlipayRootContent       []byte `json:"alipayRootContent"`
	Status                  bool   `json:"status"`
}

func (p ClientConfig) Validate() error {
	if p.AppId == "" || p.SignType == "" || p.PrivateKey == "" {
		return errorx.New(errorx.InvalidArgument, "alipay config is incomplete")
	}
	if len(p.AppPublicContent) == 0 || len(p.AlipayPublicContentRSA2) == 0 || len(p.AlipayRootContent) == 0 {
		return errorx.New(errorx.InvalidArgument, "alipay cert content is incomplete")
	}
	return nil
}

// Client 支付宝客户端，实现 model.Client 接口
type Client struct {
	Config     *ClientConfig
	client     *alipay.Client
	strategies map[string]model.PayStrategy
}

var _ model.Client = (*Client)(nil)

// NewClient 创建支付宝客户端。每次调用都返回独立实例。
func NewClient(config model.ClientConfig) *Client {
	aliConfig, ok := config.(ClientConfig)
	if !ok {
		logx.Error("config is not of type ali.ClientConfig")
		return nil
	}
	return &Client{Config: &aliConfig}
}

func (a *Client) Init() error {
	if a.Config == nil {
		return errorx.New(errorx.InvalidArgument, "alipay config is nil")
	}
	if err := a.Config.Validate(); err != nil {
		return err
	}
	// 沙箱环境：isProd=false
	client, err := alipay.NewClient(a.Config.AppId, a.Config.PrivateKey, false)
	if err != nil {
		return err
	}
	client.SetCharset(alipay.UTF8).
		SetSignType(a.Config.SignType).
		SetLocation(alipay.LocationShanghai)
	// 证书模式：自动同步验签 + 设置证书序列号
	client.AutoVerifySign(a.Config.AlipayPublicContentRSA2)
	if err := client.SetCertSnByContent(a.Config.AppPublicContent, a.Config.AlipayRootContent, a.Config.AlipayPublicContentRSA2); err != nil {
		return err
	}
	a.client = client
	a.strategies = map[string]model.PayStrategy{
		model.AlipayPc:  &pcPayStrategy{a},
		model.AlipayWap: &wapPayStrategy{a},
		model.AlipayQr:  &qrPayStrategy{a},
		model.AlipayBar: &barPayStrategy{a},
	}
	logx.Info("[init][支付宝客户端初始化完成]")
	return nil
}

func (a *Client) UnifiedOrder(ctx context.Context, code string, req model.OrderUnifiedReq) (*model.OrderResp, error) {
	if a.client == nil {
		return nil, errorx.New(errorx.InvalidArgument, "channel not available")
	}
	strategy, ok := a.strategies[code]
	if !ok {
		return nil, errorx.New(errorx.InvalidArgument, "channel not available")
	}
	return strategy.UnifiedOrder(ctx, req)
}

func (a *Client) Refresh(config model.ClientConfig) error {
	aliConfig, ok := config.(ClientConfig)
	if !ok {
		return errorx.New(errorx.InvalidArgument, "alipay config refresh failed")
	}
	a.Config = &aliConfig
	return a.Init()
}

func (a *Client) ParseOrderNotify(r []byte) (*model.OrderResp, error) {
	var notifyReq gopay.BodyMap
	if err := json.Unmarshal(r, &notifyReq); err != nil {
		return nil, err
	}
	ok, err := alipay.VerifySignWithCert(a.Config.AlipayPublicContentRSA2, notifyReq)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errorx.New(errorx.InvalidArgument, "alipay notify sign verify failed")
	}
	status := parseStatus(notifyReq.Get("trade_status"))
	if status == nil {
		return nil, errorx.New(errorx.InvalidArgument, fmt.Sprintf("unknown alipay trade status: %s", notifyReq.Get("trade_status")))
	}
	if notifyReq.Get("refund_fee") != "" {
		status = model.Ptr(model.REFUND)
	}
	return model.Of(*status, notifyReq.Get("trade_no"), model.Ptr(notifyReq.Get("buyer_user_id")),
		model.ParseDate(notifyReq.Get("gmt_payment")), notifyReq.Get("out_trade_no"), notifyReq), nil
}

// ParseRefundNotify 支付宝退款没有异步回调
func (a *Client) ParseRefundNotify([]byte) (*model.RefundResp, error) {
	return nil, errorx.New(errorx.InvalidArgument, "alipay does not support refund notify")
}

func (a *Client) UnifiedRefund(ctx context.Context, req model.RefundUnifiedReq) (*model.RefundResp, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", req.OutTradeNo).
		Set("out_request_no", req.OutRefundNo).
		Set("refund_amount", formatAmount(req.RefundPrice)).
		Set("refund_reason", req.Reason)
	resp, err := a.client.TradeRefund(ctx, bm)
	if err != nil {
		if bizErr, ok := alipay.IsBizError(err); ok {
			if bizErr.SubCode == "ACQ.SYSTEM_ERROR" || bizErr.SubCode == "SYSTEM_ERROR" {
				return &model.RefundResp{
					Status:      model.WAITING,
					OutRefundNo: req.OutRefundNo,
					RawData:     bizErr,
				}, nil
			}
			return &model.RefundResp{
				Status:           model.ERROR,
				OutRefundNo:      req.OutRefundNo,
				RawData:          bizErr,
				ChannelErrorCode: bizErr.SubCode,
				ChannelErrorMsg:  bizErr.SubMsg,
			}, nil
		}
		return nil, err
	}
	// 支付宝退款只要接口调用成功即视为退款成功，没有回调
	return &model.RefundResp{
		Status:      model.SUCCESS,
		SuccessTime: model.ParseDate(resp.Response.GmtRefundPay),
		OutRefundNo: req.OutRefundNo,
		RawData:     resp.Response,
	}, nil
}

func formatAmount(amount int32) string {
	return fmt.Sprintf("%.2f", float64(amount)/100.0)
}

func parseStatus(tradeStatus string) *uint8 {
	switch tradeStatus {
	case "WAIT_BUYER_PAY":
		return model.Ptr(model.WAITING)
	case "TRADE_FINISHED", "TRADE_SUCCESS":
		return model.Ptr(model.SUCCESS)
	case "TRADE_CLOSED":
		return model.Ptr(model.CLOSED)
	default:
		return nil
	}
}

// ParseAliClientConfig 从 JSON 配置创建支付宝配置
func ParseAliClientConfig(config string) (model.ClientConfig, error) {
	var aliClientConfig ClientConfig
	if err := json.Unmarshal([]byte(config), &aliClientConfig); err != nil {
		return nil, err
	}
	return aliClientConfig, nil
}
