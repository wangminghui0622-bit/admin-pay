package weixin

// litePayStrategy 小程序支付与公众号支付的微信侧逻辑一致，直接复用 JSAPI 策略
type litePayStrategy struct {
	pubPayStrategy
}
