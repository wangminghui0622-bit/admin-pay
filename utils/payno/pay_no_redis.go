package payno

import (
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	PayNo          = "PAY_NO:"
	OrderNoPrefix  = "P"
	RefundNoPrefix = "R"
)

// Generate 生成支付/退款单号：前缀 + yyyyMMddHHmmss + Redis 自增序号，key 60 秒过期
func Generate(db *redis.Redis, prefix string) (string, error) {
	if prefix == "" {
		prefix = OrderNoPrefix
	}
	noPrefix := prefix + time.Now().Format("20060102150405")
	key := PayNo + noPrefix
	no, err := db.Incr(key)
	if err != nil {
		return "", err
	}
	if err = db.Expire(key, 60); err != nil {
		return "", err
	}
	return noPrefix + strconv.FormatInt(no, 10), nil
}
