package logic

import (
	"context"
	_ "embed"

	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

//go:embed init.sql
var initDatabaseSQL string

const initDatabaseRedisKey = "initDatabase:pay"

type InitDatabaseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewInitDatabaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InitDatabaseLogic {
	return &InitDatabaseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *InitDatabaseLogic) InitDatabase(in *pay.Empty) (*pay.BaseResp, error) {
	exists, err := l.svcCtx.Redis.Exists(initDatabaseRedisKey)
	if err != nil {
		l.Errorf("check init database key failed: %v", err)
		return nil, errorx.New(errorx.ServerError, "redis error")
	}
	if exists {
		return &pay.BaseResp{Msg: "already initialized"}, nil
	}
	if err = l.svcCtx.DB.WithContext(l.ctx).Exec(initDatabaseSQL).Error; err != nil {
		l.Errorf("init pay database failed: %v", err)
		return nil, errorx.New(errorx.ServerError, "init database failed")
	}
	if err = l.svcCtx.Redis.Set(initDatabaseRedisKey, "1"); err != nil {
		l.Errorf("set init database key failed: %v", err)
		return nil, errorx.New(errorx.ServerError, "redis error")
	}
	return &pay.BaseResp{Msg: msgSuccess}, nil
}
