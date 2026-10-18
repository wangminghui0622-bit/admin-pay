package svc

import (
	"github.com/hibiken/asynq"
	"github.com/wangminghui0622-bit/admin-common/dal/query"
	"github.com/wangminghui0622-bit/admin-pay/internal/config"
	"github.com/wangminghui0622-bit/admin-pay/payment"
	"github.com/wangminghui0622-bit/admin-pay/payment/model"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config      config.Config
	DB          *gorm.DB
	Query       *query.PayQuery
	Redis       *redis.Redis
	AsynqClient *asynq.Client
	PayClient   map[string]model.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := gorm.Open(mysql.Open(c.DatabaseDSN), &gorm.Config{})
	logx.Must(err)

	var payConfig payment.PayConfig
	logx.Must(c.AliPayConfig.NewAliPayPayConfig(&payConfig))
	logx.Must(c.WxPayConfig.NewWxPayPayConfig(&payConfig))
	payClient, err := payment.NewClient(payConfig)
	logx.Must(err)

	redisClient := redis.MustNewRedis(c.RedisConf)

	var asynqClient *asynq.Client
	if c.AsynqConf.Enable {
		addr := c.AsynqConf.Addr
		if addr == "" {
			addr = c.RedisConf.Host
		}
		asynqClient = asynq.NewClient(asynq.RedisClientOpt{
			Addr: addr,
			DB:   c.AsynqConf.DB,
		})
	}

	return &ServiceContext{
		Config:      c,
		DB:          db,
		Query:       query.UsePay(db),
		Redis:       redisClient,
		AsynqClient: asynqClient,
		PayClient:   payClient,
	}
}
