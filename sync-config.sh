#!/bin/bash

# 配置文件同步脚本
# 用于将后端配置同步到前端

echo "🔄 同步游戏配置文件..."

# 同步序列配置
cp idlemmoserver/internal/domain/config_full.json idle-vue/src/config/sequences.json
echo "✅ 序列配置已同步"

# 同步装备配置
cp idlemmoserver/internal/domain/equipment_config.json idle-vue/src/config/equipment.json
echo "✅ 装备配置已同步"

# 验证前端配置文件
echo "📊 验证配置文件..."

if [ -f "idle-vue/src/config/sequences.json" ]; then
    echo "✅ 前端序列配置文件存在"
else
    echo "❌ 前端序列配置文件缺失"
    exit 1
fi

if [ -f "idle-vue/src/config/equipment.json" ]; then
    echo "✅ 前端装备配置文件存在"
else
    echo "❌ 前端装备配置文件缺失"
    exit 1
fi

echo "🎉 配置同步完成！"
echo ""
echo "📝 更新配置时请按以下步骤操作："
echo "   1. 修改后端配置文件 (idlemmoserver/internal/domain/)"
echo "   2. 运行此脚本: ./sync-config.sh"
echo "   3. 重启前端开发服务器"
echo "   4. 重启后端服务器"