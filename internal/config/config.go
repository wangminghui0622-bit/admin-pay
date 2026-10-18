package config

import (
	"fmt"
	"os"

	"github.com/wangminghui0622-bit/admin-pay/payment"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	DatabaseDSN   string
	RedisConf     redis.RedisConf
	AsynqConf     AsynqConf
	PayProperties model.Properties `json:",optional"`
	AliPayConfig  AliPayConfig     `json:",optional"`
	WxPayConfig   WxPayConfig      `json:",optional"`
}

// AsynqConf 仅用于入队（消费者由独立的 job 服务实现）
type AsynqConf struct {
	Addr   string `json:",default=127.0.0.1:6379"`
	DB     int    `json:",default=0"`
	Enable bool   `json:",default=true"`
}

type AliPayConfig struct {
	AppId                       string `json:",env=ALI_PAY_APPID"`                                                // appId
	SignType                    string `json:",env=ALI_PAY_SIGN_TYPE"`                                            // 签名类型
	PrivateKey                  string `json:",env=ALI_PAY_PRIVATE_KEY"`                                          // 应用私钥
	Status                      bool   `json:",default=false,env=ALI_PAY_STATUS"`                                 // 是否启用
	AppPublicContentPath        string `json:",default=cert/alipay/appPublicCert.crt,env=ALIPAY_APP_PUBLIC_CERT"` // 应用公钥证书路径
	AlipayPublicContentRSA2Path string `json:",default=cert/alipay/alipayPublicCert.crt,env=ALIPAY_PUBLIC_CERT"`  // 支付宝公钥证书路径
	AlipayRootContentPath       string `json:",default=cert/alipay/alipayRootCert.crt,env=ALIPAY_ROOT_CERT"`      // 支付宝根证书路径
}

type WxPayConfig struct {
	AppId                 string `json:",env=WX_PAY_APPID"`                                               // appId
	MchId                 string `json:",env=WX_PAY_MCHID"`                                               // 商户号
	SerialNumber          string `json:",env=WX_PAY_SERIAL_NUMBER"`                                       // apiclient_cert.pem 证书号
	ApiV3Key              string `json:",env=WX_PAY_APIV3KEY"`                                            // apiV3Key
	Status                bool   `json:",default=false,env=WX_PAY_STATUS"`                                // 是否启用
	PrivateKeyContentPath string `json:",default=cert/wechat/apiclient_key.pem,env=WX_PAY_APICLIENT_KEY"` // apiclient_key.pem 路径
}

// NewAliPayPayConfig 将支付宝配置装载为支付客户端配置，仅启用时读取证书
func (p *AliPayConfig) NewAliPayPayConfig(payConfig *payment.PayConfig) error {
	if !p.Status {
		return nil
	}
	payConfig.AliConfig.Status = true
	payConfig.AliConfig.AppId = p.AppId
	payConfig.AliConfig.SignType = p.SignType
	payConfig.AliConfig.PrivateKey = p.PrivateKey
	appPublicContent, err := readFile(p.AppPublicContentPath)
	if err != nil {
		return err
	}
	payConfig.AliConfig.AppPublicContent = appPublicContent
	alipayPublicContentRSA2, err := readFile(p.AlipayPublicContentRSA2Path)
	if err != nil {
		return err
	}
	payConfig.AliConfig.AlipayPublicContentRSA2 = alipayPublicContentRSA2
	alipayRootContent, err := readFile(p.AlipayRootContentPath)
	if err != nil {
		return err
	}
	payConfig.AliConfig.AlipayRootContent = alipayRootContent
	return nil
}

// NewWxPayPayConfig 将微信支付配置装载为支付客户端配置，仅启用时读取私钥
func (w *WxPayConfig) NewWxPayPayConfig(payConfig *payment.PayConfig) error {
	if !w.Status {
		return nil
	}
	payConfig.WxConfig.Status = true
	payConfig.WxConfig.AppId = w.AppId
	payConfig.WxConfig.MchId = w.MchId
	payConfig.WxConfig.SerialNumber = w.SerialNumber
	payConfig.WxConfig.ApiV3Key = w.ApiV3Key
	privateKeyContent, err := readFile(w.PrivateKeyContentPath)
	if err != nil {
		return err
	}
	payConfig.WxConfig.PrivateKeyContent = privateKeyContent
	return nil
}

func readFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pay cert file %s failed: %w", path, err)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("pay cert file %s is empty", path)
	}
	return content, nil
}
