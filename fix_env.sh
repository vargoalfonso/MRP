#!/bin/zsh
cp .env.example .env
echo 'DB_USERNAME=vargoalfonso' >> .env
echo 'DB_PASSWORD=' >> .env
echo 'REDIS_PASSWORD=' >> .env
echo 'JWT_ACCESS_SECRET=$(openssl rand -base64 64)' >> .env
echo 'JWT_REFRESH_SECRET=$(openssl rand -base64 64)' >> .env
source .env
echo '.env fixed!'
cat .env
