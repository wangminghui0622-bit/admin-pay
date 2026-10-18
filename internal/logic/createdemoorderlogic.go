package logic

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/wangminghui0622-bit/admin-common/dal/model"
	"github.com/wangminghui0622-bit/admin-common/errorx"
	"github.com/wangminghui0622-bit/admin-pay/internal/svc"
	"github.com/wangminghui0622-bit/admin-pay/pay"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDemoOrderLogic struct {
	ctx      context.Context
	svcCtx   *svc.ServiceContext
	spuNames map[uint64]spu
	logx.Logger
}

type spu struct {
	name  string
	price int32
}

func NewCreateDemoOrderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDemoOrderLogic {
	spuNames := map[uint64]spu{
		1: {name: "华为手机", price: 1},
		2: {name: "小米电视", price: 10},
		3: {name: "苹果手机", price: 100},
		4: {name: "华硕笔记本", price: 1000},
		5: {name: "蔚来汽车", price: 200000},
	}
	return &CreateDemoOrderLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		spuNames: spuNames,
		Logger:   logx.WithContext(ctx),
	}
}

func (l *CreateDemoOrderLogic) CreateDemoOrder(in *pay.PayDemoOrderCreateReq) (*pay.BaseIDResp, error) {
	spuInfo, ok := l.spuNames[in.SpuId]
	if !ok {
		return nil, errorx.New(errorx.InvalidArgument, fmt.Sprintf("spuId：%d 商品不存在", in.SpuId))
	}
	refundPrice := int32(0)
	demoOrder := &model.PayDemoOrder{
		UserID:      in.UserId,
		SpuID:       int64(in.SpuId),
		SpuName:     spuInfo.name,
		Price:       spuInfo.price,
		PayStatus:   false,
		RefundPrice: &refundPrice,
	}
	if err := l.svcCtx.Query.PayDemoOrder.WithContext(l.ctx).Create(demoOrder); err != nil {
		return nil, dbError(l.Logger, err, in)
	}

	payOrder, err := NewCreateOrderLogic(l.ctx, l.svcCtx).CreateOrder(&pay.OrderCreateReq{
		UserIp:          in.UserIp,
		MerchantOrderId: strconv.FormatInt(demoOrder.ID, 10),
		Subject:         spuInfo.name,
		Body:            spuInfo.name,
		Price:           spuInfo.price,
		ExpireTime:      time.Now().Add(2 * time.Hour).Unix(),
	})
	if err != nil {
		return nil, err
	}
	result := l.svcCtx.DB.WithContext(l.ctx).Model(&model.PayDemoOrder{}).
		Where("id = ?", demoOrder.ID).
		Updates(map[string]any{"pay_order_id": int64(payOrder.Id)})
	if result.Error != nil {
		return nil, dbError(l.Logger, result.Error, in)
	}
	return &pay.BaseIDResp{Id: uint64(demoOrder.ID), Msg: msgSuccess}, nil
}
