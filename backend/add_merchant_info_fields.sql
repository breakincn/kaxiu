-- 为 merchants 表添加营业时间和地址信息字段
ALTER TABLE merchants
ADD COLUMN morning_start VARCHAR(10) DEFAULT '' COMMENT '上午营业开始时间（格式：HH:MM）' AFTER avg_service_minutes,
ADD COLUMN morning_end VARCHAR(10) DEFAULT '' COMMENT '上午营业结束时间（格式：HH:MM）' AFTER morning_start,
ADD COLUMN afternoon_start VARCHAR(10) DEFAULT '' COMMENT '下午营业开始时间（格式：HH:MM）' AFTER morning_end,
ADD COLUMN afternoon_end VARCHAR(10) DEFAULT '' COMMENT '下午营业结束时间（格式：HH:MM）' AFTER afternoon_start,
ADD COLUMN evening_start VARCHAR(10) DEFAULT '' COMMENT '晚上营业开始时间（格式：HH:MM）' AFTER afternoon_end,
ADD COLUMN evening_end VARCHAR(10) DEFAULT '' COMMENT '晚上营业结束时间（格式：HH:MM）' AFTER evening_start,
ADD COLUMN all_day_start VARCHAR(10) DEFAULT '' COMMENT '全天营业开始时间（格式：HH:MM）' AFTER evening_end,
ADD COLUMN all_day_end VARCHAR(10) DEFAULT '' COMMENT '全天营业结束时间（格式：HH:MM）' AFTER all_day_start,
ADD COLUMN province VARCHAR(50) DEFAULT '' COMMENT '省份' AFTER all_day_end,
ADD COLUMN city VARCHAR(50) DEFAULT '' COMMENT '城市' AFTER province,
ADD COLUMN district VARCHAR(50) DEFAULT '' COMMENT '区县' AFTER city,
ADD COLUMN address VARCHAR(200) DEFAULT '' COMMENT '详细地址（街道门牌号）' AFTER district;
