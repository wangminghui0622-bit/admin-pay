-- admin-zero pay 模块初始化脚本
-- 可安全重复执行：建表用 CREATE TABLE IF NOT EXISTS，数据用 INSERT IGNORE。

-- ----------------------------
-- 支付订单表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `pay_order` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create Time | 创建日期',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update Time | 修改日期',
  `status` tinyint unsigned DEFAULT '1' COMMENT 'Status 1: normal 2: ban | 状态 1 正常 2 禁用',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time | 删除日期',
  `channel_code` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '渠道编码',
  `merchant_order_id` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '商户订单编号',
  `subject` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '商品标题',
  `body` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '商品描述',
  `price` int NOT NULL COMMENT '支付金额，单位：分',
  `channel_fee_rate` float DEFAULT NULL COMMENT '渠道手续费，单位：百分比',
  `channel_fee_price` int DEFAULT NULL COMMENT '渠道手续金额，单位：分',
  `user_ip` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '用户 IP',
  `expire_time` datetime NOT NULL COMMENT '订单失效时间',
  `success_time` datetime DEFAULT NULL COMMENT '订单支付成功时间',
  `notify_time` datetime DEFAULT NULL COMMENT '订单支付通知时间',
  `extension_id` bigint unsigned DEFAULT NULL COMMENT '支付成功的订单拓展单编号',
  `no` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '订单号',
  `refund_price` int NOT NULL COMMENT '退款总金额，单位：分',
  `channel_user_id` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '渠道用户编号',
  `channel_order_no` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '渠道订单号',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='支付订单表';

-- ----------------------------
-- 支付订单拓展表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `pay_order_extension` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create Time | 创建日期',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update Time | 修改日期',
  `status` tinyint unsigned DEFAULT '1' COMMENT 'Status 1: normal 2: ban | 状态 1 正常 2 禁用',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time | 删除日期',
  `no` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '支付订单号',
  `order_id` bigint unsigned NOT NULL COMMENT '支付订单编号 pay_order 表id',
  `channel_code` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '渠道编码',
  `user_ip` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '用户 IP',
  `channel_extras` json DEFAULT NULL COMMENT '支付渠道的额外参数',
  `channel_error_code` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '调用渠道的错误码',
  `channel_error_msg` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '调用渠道报错时，错误信息',
  `channel_notify_data` text COLLATE utf8mb4_bin COMMENT '支付渠道异步通知的内容',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='支付订单拓展表';

-- ----------------------------
-- 支付退款表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `pay_refund` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create Time | 创建日期',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update Time | 修改日期',
  `status` tinyint unsigned DEFAULT '1' COMMENT 'Status 1: normal 2: ban | 状态 1 正常 2 禁用',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time | 删除日期',
  `no` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '退款单号',
  `channel_code` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '渠道编码',
  `order_id` bigint unsigned NOT NULL COMMENT '支付订单编号 pay_order 表id',
  `order_no` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '支付订单 no',
  `merchant_order_id` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '商户订单编号（商户系统生成）',
  `merchant_refund_id` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '商户退款订单号（商户系统生成）',
  `pay_price` int NOT NULL COMMENT '支付金额,单位分',
  `refund_price` int NOT NULL COMMENT '退款金额,单位分',
  `reason` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '退款原因',
  `user_ip` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '用户 IP',
  `channel_order_no` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '渠道订单号，pay_order 中的 channel_order_no 对应',
  `channel_refund_no` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '渠道退款单号，渠道返回',
  `success_time` datetime DEFAULT NULL COMMENT '退款成功时间',
  `channel_error_code` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '渠道调用报错时，错误码',
  `channel_error_msg` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '渠道调用报错时，错误信息',
  `channel_notify_data` text COLLATE utf8mb4_bin COMMENT '支付渠道异步通知的内容',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='支付退款表';

-- ----------------------------
-- 支付示例订单表
-- ----------------------------
CREATE TABLE IF NOT EXISTS `pay_demo_order` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Create Time | 创建日期',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update Time | 修改日期',
  `deleted_at` datetime DEFAULT NULL COMMENT 'Delete Time | 删除日期',
  `user_id` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '用户编号',
  `spu_id` bigint unsigned NOT NULL COMMENT '商品编号',
  `spu_name` varchar(255) COLLATE utf8mb4_bin NOT NULL COMMENT '商品名称',
  `price` int NOT NULL COMMENT '价格，单位：分',
  `pay_status` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否支付',
  `pay_order_id` bigint unsigned DEFAULT NULL COMMENT '支付订单编号',
  `pay_time` datetime DEFAULT NULL COMMENT '付款时间',
  `pay_channel_code` varchar(255) COLLATE utf8mb4_bin DEFAULT NULL COMMENT '支付渠道',
  `pay_refund_id` bigint unsigned DEFAULT NULL COMMENT '支付退款单号',
  `refund_price` int DEFAULT NULL COMMENT '退款金额，单位：分',
  `refund_time` datetime DEFAULT NULL COMMENT '退款完成时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='支付示例订单表';

-- ----------------------------
-- API 注册（id 5001-6000，service_name = Pay）
-- ----------------------------
INSERT IGNORE INTO `sys_apis` (`id`, `created_at`, `updated_at`, `path`, `description`, `api_group`, `service_name`, `method`, `is_required`) VALUES
(5001, NOW(), NOW(), '/pay-api/init/database', '初始化支付数据库', 'Pay', 'Pay', 'POST', 0),
(5002, NOW(), NOW(), '/pay-api/demo_order/create', '创建示例订单', 'PayDemoOrder', 'Pay', 'POST', 0),
(5003, NOW(), NOW(), '/pay-api/demo_order/get', '获取示例订单', 'PayDemoOrder', 'Pay', 'POST', 0),
(5004, NOW(), NOW(), '/pay-api/demo_order/list', '示例订单列表', 'PayDemoOrder', 'Pay', 'POST', 0),
(5005, NOW(), NOW(), '/pay-api/demo_order/update_paid', '更新示例订单支付状态', 'PayDemoOrder', 'Pay', 'POST', 0),
(5006, NOW(), NOW(), '/pay-api/demo_order/refund', '示例订单退款', 'PayDemoOrder', 'Pay', 'POST', 0),
(5007, NOW(), NOW(), '/pay-api/demo_order/update_refund_paid', '更新示例订单退款状态', 'PayDemoOrder', 'Pay', 'POST', 0),
(5008, NOW(), NOW(), '/pay-api/order/create', '创建支付订单', 'PayOrder', 'Pay', 'POST', 0),
(5009, NOW(), NOW(), '/pay-api/order/get', '获取支付订单', 'PayOrder', 'Pay', 'POST', 0),
(5010, NOW(), NOW(), '/pay-api/order/list', '支付订单分页', 'PayOrder', 'Pay', 'POST', 0),
(5011, NOW(), NOW(), '/pay-api/order/extension/create', '创建支付订单拓展', 'PayOrder', 'Pay', 'POST', 0),
(5012, NOW(), NOW(), '/pay-api/order/extension/get', '获取支付订单拓展', 'PayOrder', 'Pay', 'POST', 0),
(5013, NOW(), NOW(), '/pay-api/order/notify', '支付渠道回调', 'PayOrder', 'Pay', 'POST', 0),
(5014, NOW(), NOW(), '/pay-api/order/submit', '提交支付', 'PayOrder', 'Pay', 'POST', 0),
(5015, NOW(), NOW(), '/pay-api/refund/create', '创建退款', 'PayRefund', 'Pay', 'POST', 0),
(5016, NOW(), NOW(), '/pay-api/refund/list', '退款分页', 'PayRefund', 'Pay', 'POST', 0),
(5017, NOW(), NOW(), '/pay-api/refund/get', '获取退款', 'PayRefund', 'Pay', 'POST', 0),
(5018, NOW(), NOW(), '/pay-api/refund/notify', '退款渠道回调', 'PayRefund', 'Pay', 'POST', 0);

-- ----------------------------
-- casbin 授权（id 10001+，授权超级管理员角色 001）
-- ----------------------------
INSERT IGNORE INTO `casbin_rules` (`id`, `ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`) VALUES
(10001, 'p', '001', '/pay-api/init/database', 'POST', '', '', ''),
(10002, 'p', '001', '/pay-api/demo_order/create', 'POST', '', '', ''),
(10003, 'p', '001', '/pay-api/demo_order/get', 'POST', '', '', ''),
(10004, 'p', '001', '/pay-api/demo_order/list', 'POST', '', '', ''),
(10005, 'p', '001', '/pay-api/demo_order/update_paid', 'POST', '', '', ''),
(10006, 'p', '001', '/pay-api/demo_order/refund', 'POST', '', '', ''),
(10007, 'p', '001', '/pay-api/demo_order/update_refund_paid', 'POST', '', '', ''),
(10008, 'p', '001', '/pay-api/order/create', 'POST', '', '', ''),
(10009, 'p', '001', '/pay-api/order/get', 'POST', '', '', ''),
(10010, 'p', '001', '/pay-api/order/list', 'POST', '', '', ''),
(10011, 'p', '001', '/pay-api/order/extension/create', 'POST', '', '', ''),
(10012, 'p', '001', '/pay-api/order/extension/get', 'POST', '', '', ''),
(10013, 'p', '001', '/pay-api/order/notify', 'POST', '', '', ''),
(10014, 'p', '001', '/pay-api/order/submit', 'POST', '', '', ''),
(10015, 'p', '001', '/pay-api/refund/create', 'POST', '', '', ''),
(10016, 'p', '001', '/pay-api/refund/list', 'POST', '', '', ''),
(10017, 'p', '001', '/pay-api/refund/get', 'POST', '', '', ''),
(10018, 'p', '001', '/pay-api/refund/notify', 'POST', '', '', '');
