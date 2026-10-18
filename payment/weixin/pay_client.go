package weixin

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
	"github.com/zeromicro/go-zero/core/logx"
)

// ClientConfig 微信支付 V3 配置，实现 model.ClientConfig 接口。
// 协议说明：https://pay.weixin.qq.com/wiki/doc/apiv3/wechatpay/wechatpay-1.shtml
type ClientConfig struct {
	AppId             string `json:"appId"`             // appId
	MchId             string `json:"mchId"`             // 商户号
	PrivateKeyContent []byte `json:"privateKeyContent"` // apiclient_key.pem 内容
	SerialNumber      string `json:"serialNumber"`      // apiclient_cert.pem 证书号
	ApiV3Key          string `json:"apiV3Key"`          // apiV3Key
	Status            bool   `json:"status"`            // 是否启用
}

func (p ClientConfig) Validate() error {
	if p.AppId == "" || p.MchId == "" || p.SerialNumber == "" || p.ApiV3Key == "" || len(p.PrivateKeyContent) == 0 {
		return errorx.New(errorx.InvalidArgument, "wechat pay config is incomplete")
	}
	return nil
}

type Client struct {
	Config     *ClientConfig
	client     *wechat.ClientV3
	strategies map[string]model.PayStrategy
}

var _ model.Client = (*Client)(nil)

// NewClient 创建微信支付客户端。每次调用都返回独立实例，避免多商户配置被单例污染。
func NewClient(config model.ClientConfig) *Client {
	wxConfig, ok := config.(ClientConfig)
	if !ok {
		logx.Error("config is not of type weixin.ClientConfig")
		return nil
	}
	return &Client{Config: &wxConfig}
}

func (w *Client) Init() error {
	if w.Config == nil {
		return errorx.New(errorx.InvalidArgument, "wechat pay config is nil")
	}
	if err := w.Config.Validate(); err != nil {
		return err
	}
	client, err := wechat.NewClientV3(w.Config.MchId, w.Config.SerialNumber, w.Config.ApiV3Key, string(w.Config.PrivateKeyContent))
	if err != nil {
		return err
	}
	// 启用自动同步返回验签，并定时更新微信平台 API 证书
	if err := client.AutoVerifySign(); err != nil {
		return err
	}
	w.client = client
	w.strategies = map[string]model.PayStrategy{
		model.WxApp:    &appPayStrategy{w},
		model.WxPub:    &pubPayStrategy{w},
		model.WxNative: &nativePayStrategy{w},
		model.WxLite:   &litePayStrategy{pubPayStrategy{w}},
	}
	logx.Info("[init][微信支付客户端初始化完成]")
	return nil
}

func (w *Client) UnifiedOrder(ctx context.Context, code string, req model.OrderUnifiedReq) (*model.OrderResp, error) {
	if w.client == nil {
		return nil, errorx.New(errorx.InvalidArgument, "channel not available")
	}
	strategy, ok := w.strategies[code]
	if !ok {
		return nil, errorx.New(errorx.InvalidArgument, "channel not available")
	}
	return strategy.UnifiedOrder(ctx, req)
}

func (w *Client) Refresh(config model.ClientConfig) error {
	wxConfig, ok := config.(ClientConfig)
	if !ok {
		return errorx.New(errorx.InvalidArgument, "wechat pay config refresh failed")
	}
	w.Config = &wxConfig
	return w.Init()
}

func (w *Client) ParseOrderNotify(r []byte) (*model.OrderResp, error) {
	req := new(wechat.V3NotifyReq)
	if err := json.Unmarshal(r, req); err != nil {
		return nil, err
	}
	cert := w.client.WxPublicKey()
	if cert == nil {
		return nil, errorx.New(errorx.ServerError, "wechat platform certificate is not ready")
	}
	if err := req.VerifySignByPK(cert); err != nil {
		return nil, err
	}
	result, err := req.DecryptCipherText(string(w.client.ApiV3Key))
	if err != nil {
		return nil, err
	}
	status, err := parseStatus(result.TradeState)
	if err != nil {
		return nil, err
	}
	var openid *string
	if result.Payer != nil {
		openid = &result.Payer.Openid
	}
	return model.Of(status, result.TransactionId, openid, model.ParseDate(result.SuccessTime), result.OutTradeNo, req), nil
}

func (w *Client) UnifiedRefund(ctx context.Context, req model.RefundUnifiedReq) (*model.RefundResp, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", req.OutTradeNo).
		Set("out_refund_no", req.OutRefundNo).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("refund", req.RefundPrice).
				Set("total", req.PayPrice).
				Set("currency", "CNY")
		}).
		Set("reason", req.Reason).
		Set("notify_url", req.NotifyUrl)

	resp, err := w.client.V3Refund(ctx, bm)
	if err != nil {
		return nil, err
	}
	if resp.Code != wechat.Success {
		return &model.RefundResp{
			Status:           model.ERROR,
			OutRefundNo:      req.OutRefundNo,
			RawData:          resp,
			ChannelErrorCode: strconv.Itoa(resp.Code),
			ChannelErrorMsg:  resp.Error,
		}, nil
	}
	switch resp.Response.Status {
	case "SUCCESS":
		return &model.RefundResp{
			Status:          model.SUCCESS,
			ChannelRefundNo: resp.Response.RefundId,
			SuccessTime:     model.ParseDate(resp.Response.SuccessTime),
			OutRefundNo:     resp.Response.OutRefundNo,
			RawData:         resp.Response,
		}, nil
	case "PROCESSING":
		return &model.RefundResp{
			Status:          model.WAITING,
			ChannelRefundNo: resp.Response.RefundId,
			OutRefundNo:     resp.Response.OutRefundNo,
			RawData:         resp.Response,
		}, nil
	default:
		return &model.RefundResp{
			Status:          model.ERROR,
			OutRefundNo:     resp.Response.OutRefundNo,
			RawData:         resp.Response,
			ChannelErrorMsg: resp.Response.Status,
		}, nil
	}
}

func (w *Client) ParseRefundNotify(r []byte) (*model.RefundResp, error) {
	req := new(wechat.V3NotifyReq)
	if err := json.Unmarshal(r, req); err != nil {
		return nil, err
	}
	cert := w.client.WxPublicKey()
	if cert == nil {
		return nil, errorx.New(errorx.ServerError, "wechat platform certificate is not ready")
	}
	if err := req.VerifySignByPK(cert); err != nil {
		return nil, err
	}
	result, err := req.DecryptRefundCipherText(string(w.client.ApiV3Key))
	if err != nil {
		return nil, err
	}
	switch result.RefundStatus {
	case "SUCCESS":
		return &model.RefundResp{
			Status:          model.SUCCESS,
			ChannelRefundNo: result.RefundId,
			SuccessTime:     model.ParseDate(result.SuccessTime),
			OutRefundNo:     result.OutRefundNo,
			RawData:         result,
		}, nil
	case "PROCESSING":
		return &model.RefundResp{
			Status:          model.WAITING,
			ChannelRefundNo: result.RefundId,
			OutRefundNo:     result.OutRefundNo,
			RawData:         result,
		}, nil
	default:
		return &model.RefundResp{
			Status:           model.ERROR,
			ChannelRefundNo:  result.RefundId,
			OutRefundNo:      result.OutRefundNo,
			RawData:          result,
			ChannelErrorCode: result.RefundStatus,
			ChannelErrorMsg:  result.RefundStatus,
		}, nil
	}
}

// buildPayUnifiedOrderBm 构造下单公共参数
func (w *Client) buildPayUnifiedOrderBm(req model.OrderUnifiedReq) gopay.BodyMap {
	bm := make(gopay.BodyMap)
	bm.Set("appid", w.Config.AppId).
		Set("description", req.Subject).
		Set("out_trade_no", req.OutTradeNo).
		Set("time_expire", req.ExpireTime.Format(time.RFC3339)).
		Set("notify_url", req.NotifyUrl).
		SetBodyMap("scene_info", func(bm gopay.BodyMap) {
			bm.Set("payer_client_ip", req.UserIp)
		}).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("total", req.Price).
				Set("currency", "CNY")
		})
	return bm
}

// parseStatus 微信交易状态映射
func parseStatus(tradeState string) (uint8, error) {
	switch tradeState {
	case "NOTPAY", "USERPAYING":
		return model.WAITING, nil
	case "SUCCESS":
		return model.SUCCESS, nil
	case "REFUND":
		return model.REFUND, nil
	case "CLOSED", "REVOKED", "PAYERROR":
		return model.CLOSED, nil
	default:
		return model.ERROR, errorx.New(errorx.InvalidArgument, fmt.Sprintf("unknown wechat trade state: %s", tradeState))
	}
}
