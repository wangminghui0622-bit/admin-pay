package payment

import (
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/payment/ali"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
	"github.com/wangminghui0622-bit/admin-pay/payment/weixin"
)

// PayConfig 支付渠道配置集合
type PayConfig struct {
	AliConfig ali.ClientConfig
	WxConfig  weixin.ClientConfig
}

// NewClient 按启用的渠道创建支付客户端映射。
// Status=false 的渠道不会出现在映射中；Status=true 但配置/证书不完整时返回错误，
// 由调用方在启动阶段显式失败，避免渠道静默缺席。
func NewClient(c PayConfig) (map[string]model.Client, error) {
	payClient := make(map[string]model.Client)

	if c.AliConfig.Status {
		if err := c.AliConfig.Validate(); err != nil {
			return nil, err
		}
		aliClient := ali.NewClient(c.AliConfig)
		if aliClient == nil {
			return nil, errorx.New(errorx.ServerError, "create alipay client failed")
		}
		if err := aliClient.Init(); err != nil {
			return nil, err
		}
		for _, code := range []string{model.AlipayPc, model.AlipayWap, model.AlipayQr, model.AlipayBar} {
			payClient[code] = aliClient
		}
	}

	if c.WxConfig.Status {
		if err := c.WxConfig.Validate(); err != nil {
			return nil, err
		}
		wxClient := weixin.NewClient(c.WxConfig)
		if wxClient == nil {
			return nil, errorx.New(errorx.ServerError, "create wechat pay client failed")
		}
		if err := wxClient.Init(); err != nil {
			return nil, err
		}
		for _, code := range []string{model.WxPub, model.WxApp, model.WxNative, model.WxLite} {
			payClient[code] = wxClient
		}
	}

	return payClient, nil
}

// GetClient 安全获取支付客户端，渠道未启用时返回明确的业务错误而不是 nil panic
func GetClient(clients map[string]model.Client, channelCode string) (model.Client, error) {
	if clients == nil {
		return nil, errorx.New(errorx.InvalidArgument, "channel not available")
	}
	client, ok := clients[channelCode]
	if !ok || client == nil {
		return nil, errorx.New(errorx.InvalidArgument, "channel not available")
	}
	return client, nil
}
