# Verification API Service Makefile
# 验证API服务管理脚本

.PHONY: help build deploy test-api

# 默认目标
.DEFAULT_GOAL := help

# 项目配置
SERVICE_NAME := verification-api-service
CONTAINER_NAME := verification-api-service
PORT := 8080
API_BASE_URL := http://localhost:$(PORT)

# 颜色定义
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

help: ## 显示帮助信息
	@echo "$(GREEN)Verification API Service 管理命令$(NC)"
	@echo ""
	@echo "$(YELLOW)构建和部署:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "(build|deploy)"
	@echo ""
	@echo "$(YELLOW)测试命令:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "(test-)"
	@echo ""
	@echo "$(YELLOW)监控命令:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "(monitor)"
	@echo ""

# 构建和部署命令
build: ## 编译代码检查错误
	@echo "$(GREEN)编译代码...$(NC)"
	@go build -o /dev/null ./cmd/server
	@echo "$(GREEN)编译成功，没有错误！$(NC)"

deploy: ## 部署到 Railway
	@echo "$(GREEN)部署到 Railway...$(NC)"
	@if ! command -v railway > /dev/null; then \
		echo "$(RED)错误: Railway CLI 未安装$(NC)"; \
		echo "$(YELLOW)请先安装 Railway CLI: https://docs.railway.app/develop/cli$(NC)"; \
		exit 1; \
	fi
	@railway up
	@echo "$(GREEN)部署完成！$(NC)"

# 测试命令
test-api: ## 测试 API 接口
	@echo "$(GREEN)测试 API 接口...$(NC)"
	@if [ ! -f test_api.sh ]; then \
		echo "$(RED)错误: test_api.sh 不存在$(NC)"; \
		exit 1; \
	fi
	chmod +x test_api.sh
	./test_api.sh

test-brevo: ## 测试 Brevo SDK
	@echo "$(GREEN)测试 Brevo SDK...$(NC)"
	@if [ ! -f test_brevo_sdk_fixed.sh ]; then \
		echo "$(RED)错误: test_brevo_sdk_fixed.sh 不存在$(NC)"; \
		exit 1; \
	fi
	chmod +x test_brevo_sdk_fixed.sh
	./test_brevo_sdk_fixed.sh

test-health: ## 测试健康检查
	@echo "$(GREEN)测试健康检查...$(NC)"
	@curl -s $(API_BASE_URL)/health | jq . || echo "$(RED)健康检查失败$(NC)"

test-send: ## 发送测试邮件
	@echo "$(GREEN)发送测试邮件...$(NC)"
	@read -p "请输入邮箱地址: " email; \
	curl -X POST $(API_BASE_URL)/api/verification/send-code \
		-H "Content-Type: application/json" \
		-H "X-Project-ID: default" \
		-H "X-API-Key: default-api-key" \
		-d "{\"email\": \"$$email\", \"project_id\": \"default\"}" | jq .


# 监控命令
monitor: ## 监控服务状态
	@echo "$(GREEN)监控服务状态...$(NC)"
	@while true; do \
		clear; \
		echo "$(GREEN)=== Auth-Mail Service 状态 ===$(NC)"; \
		echo ""; \
		echo "$(YELLOW)服务状态:$(NC)"; \
		docker-compose ps; \
		echo ""; \
		echo "$(YELLOW)健康检查:$(NC)"; \
		curl -s $(API_BASE_URL)/health | jq . 2>/dev/null || echo "$(RED)服务不可用$(NC)"; \
		echo ""; \
		echo "$(YELLOW)按 Ctrl+C 退出监控$(NC)"; \
		sleep 5; \
	done


# 环境命令
env-check: ## 检查环境变量
	@echo "$(GREEN)检查环境变量...$(NC)"
	@echo "$(YELLOW)必需的环境变量:$(NC)"
	@echo "  BREVO_API_KEY: $(if $(BREVO_API_KEY),$(GREEN)已设置$(NC),$(RED)未设置$(NC))"
	@echo "  BREVO_FROM_EMAIL: $(if $(BREVO_FROM_EMAIL),$(GREEN)已设置$(NC),$(RED)未设置$(NC))"
	@echo "  BREVO_FROM_NAME: $(if $(BREVO_FROM_NAME),$(GREEN)已设置$(NC),$(RED)未设置$(NC))"
	@echo "  DATABASE_URL: $(if $(DATABASE_URL),$(GREEN)已设置$(NC),$(RED)未设置$(NC))"
	@echo "  REDIS_URL: $(if $(REDIS_URL),$(GREEN)已设置$(NC),$(RED)未设置$(NC))"


# 完整测试命令
test-all: ## 运行所有测试
	@echo "$(GREEN)运行完整测试套件...$(NC)"
	@make test-health
	@make test-brevo
	@make test-api
	@echo "$(GREEN)所有测试完成！$(NC)"

# 项目管理命令
project-list: ## 列出所有项目
	@echo "$(GREEN)获取项目列表...$(NC)"
	@curl -s $(API_BASE_URL)/api/admin/projects | jq .

project-create: ## 创建新项目（交互式）
	@echo "$(GREEN)创建新项目...$(NC)"
	@read -p "项目ID: " project_id; \
	read -p "项目名称: " project_name; \
	read -p "API密钥: " api_key; \
	read -p "发件人邮箱: " from_email; \
	read -p "发件人名称: " from_name; \
	read -p "项目描述: " description; \
	curl -X POST $(API_BASE_URL)/api/admin/projects \
		-H "Content-Type: application/json" \
		-d "{\"project_id\": \"$$project_id\", \"project_name\": \"$$project_name\", \"api_key\": \"$$api_key\", \"from_email\": \"$$from_email\", \"from_name\": \"$$from_name\", \"description\": \"$$description\", \"rate_limit\": 60, \"max_requests\": 1000}" | jq .

project-stats: ## 查看项目统计（交互式）
	@echo "$(GREEN)查看项目统计...$(NC)"
	@read -p "项目ID: " project_id; \
	curl -s $(API_BASE_URL)/api/admin/projects/$$project_id/stats | jq .
