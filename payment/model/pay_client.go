package model

import (
	"context"
	"time"
)

// 支付展示方式
const (
	Url       = "url"
	Iframe    = "iframe"
	From      = "from"
	QrCode    = "qr_code"
	QrCodeUrl = "qr_code_url"
	BarCode   = "bar_code"
	App       = "app"
)

// 支付状态，取值与 proto PayStatus 保持一致
const (
	WAITING uint8 = iota + 1
	SUCCESS
	REFUND
	CLOSED
	ERROR
)

// 支付渠道
const (
	Wx       = "Wx"
	WxPub    = "WxPub"
	WxLite   = "WxLite"
	WxApp    = "WxApp"
	WxNative = "WxNative"

	Ali       = "Ali"
	AlipayPc  = "AlipayPc"
	AlipayWap = "AlipayWap"
	AlipayQr  = "AlipayQr"
	AlipayBar = "AlipayBar"
	Mock      = "mock"
)

type (
	// ClientConfig 支付客户端配置
	ClientConfig interface {
		// Validate 参数校验
		Validate() error
	}

	// Properties 支付配置
	Properties struct {
		OrderNotifyUrl  string `json:",env=ORDER_NOTIFY_URL"`
		RefundNotifyUrl string `json:",env=REFUND_NOTIFY_URL"`
		OrderNoPrefix   string `json:",default=P"`
		RefundNoPrefix  string `json:",default=R"`
	}

	// PayStrategy 渠道内的具体支付方式（Native/JSAPI/App/PC/Wap/Qr/Bar）
	PayStrategy interface {
		UnifiedOrder(ctx context.Context, req OrderUnifiedReq) (*OrderResp, error)
	}

	// Client 支付客户端
	Client interface {
		// Init 初始化
		Init() error
		// UnifiedOrder 统一下单
		UnifiedOrder(context.Context, string, OrderUnifiedReq) (*OrderResp, error)
		// Refresh 刷新配置
		Refresh(config ClientConfig) error
		// UnifiedRefund 退款，返回 WAITING 时依赖渠道异步通知
		UnifiedRefund(context.Context, RefundUnifiedReq) (*RefundResp, error)
		// ParseOrderNotify 解析支付回调
		ParseOrderNotify([]byte) (*OrderResp, error)
		// ParseRefundNotify 解析退款回调
		ParseRefundNotify([]byte) (*RefundResp, error)
	}

	OrderUnifiedReq struct {
		UserIp        string
		OutTradeNo    string
		Subject       string
		Body          string
		NotifyUrl     string
		ReturnUrl     string
		Price         int32
		ExpireTime    time.Time
		ChannelExtras map[string]string
		DisplayMode   string
	}

	OrderResp struct {
		Status           uint8
		OutTradeNo       string
		ChannelOrderNo   string
		ChannelUserId    *string
		SuccessTime      time.Time
		RawData          any
		DisplayMode      *string
		DisplayContent   *string
		ChannelErrorCode *string
		ChannelErrorMsg  *string
	}

	RefundUnifiedReq struct {
		OutTradeNo  string
		OutRefundNo string
		Reason      string
		PayPrice    int32
		RefundPrice int32
		NotifyUrl   string
	}

	RefundResp struct {
		Status           uint8
		OutRefundNo      string
		ChannelRefundNo  string
		SuccessTime      time.Time
		RawData          any
		ChannelErrorCode string
		ChannelErrorMsg  string
	}
)

// Ptr 返回任意值的指针
func Ptr[T any](v T) *T {
	return &v
}

// WaitingOf 创建等待支付的订单结果
func WaitingOf(displayMode, displayContent *string, outTradeNo string, rawData any) *OrderResp {
	return &OrderResp{
		Status:         WAITING,
		DisplayMode:    displayMode,
		DisplayContent: displayContent,
		OutTradeNo:     outTradeNo,
		RawData:        rawData,
	}
}

// SuccessOf 创建支付成功的订单结果
func SuccessOf(channelOrderNo string, channelUserId string, successTime time.Time, outTradeNo string, rawData any) *OrderResp {
	return &OrderResp{
		Status:         SUCCESS,
		ChannelOrderNo: channelOrderNo,
		ChannelUserId:  Ptr(channelUserId),
		SuccessTime:    successTime,
		OutTradeNo:     outTradeNo,
		RawData:        rawData,
	}
}

// Of 创建指定状态的订单结果
func Of(status uint8, channelOrderNo string, channelUserId *string, successTime time.Time, outTradeNo string, rawData any) *OrderResp {
	return &OrderResp{
		Status:         status,
		ChannelOrderNo: channelOrderNo,
		ChannelUserId:  channelUserId,
		SuccessTime:    successTime,
		OutTradeNo:     outTradeNo,
		RawData:        rawData,
	}
}

// CloseOf 创建已关闭的订单结果，并带上渠道错误信息
func CloseOf(channelErrorCode, channelErrorMsg, outTradeNo string, rawData any) *OrderResp {
	return &OrderResp{
		Status:           CLOSED,
		ChannelErrorCode: Ptr(channelErrorCode),
		ChannelErrorMsg:  Ptr(channelErrorMsg),
		OutTradeNo:       outTradeNo,
		RawData:          rawData,
	}
}

// ParseDate 解析渠道时间字符串，兼容 RFC3339 与支付宝的 "2006-01-02 15:04:05"
func ParseDate(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05+08:00",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, timeStr); err == nil {
			return t
		}
	}
	// 渠道时间格式未知时，退化为当前时间，避免丢失成功时间
	return time.Now()
}
