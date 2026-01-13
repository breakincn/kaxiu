#!/bin/bash

echo "移除证书信任..."

# 移除根CA证书（需要管理员密码）
echo "移除mkcert根CA证书..."
sudo security delete-certificate -c "mkcert will@10.0.0.20" /Library/Keychains/System.keychains 2>/dev/null || true
security delete-certificate -c "mkcert will@10.0.0.20" ~/Library/Keychains/login.keychain-db 2>/dev/null || true

# 移除单个证书
echo "移除项目证书..."
security delete-certificate -c "10.0.0.20" ~/Library/Keychains/login.keychain-db 2>/dev/null || true

echo "证书信任已移除！"
echo "建议：开发完成后运行此脚本清理证书。"
