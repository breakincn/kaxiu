#!/bin/bash

echo "仅安装当前项目证书（更安全）..."

# 仅添加当前项目的证书到用户钥匙串
security add-trusted-cert -d -r trustAsUser -k ~/Library/Keychains/login.keychain-db ssl/cert.pem

echo "当前证书已安装！"
echo "注意：这只信任 10.0.0.20 的证书，不会信任其他mkcert证书。"
echo "如需完全移除，请在钥匙串访问中删除相关证书。"
