# Verification API Service Makefile
# 验证API服务管理脚本

.PHONY: help build up down logs restart status test-api

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
	@echo "$(YELLOW)服务命令:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "(build|up|down|restart|status|logs)"
	@echo ""
	@echo "$(YELLOW)测试命令:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "(test-)"
	@echo ""
	@echo "$(YELLOW)监控命令:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | grep -E "(monitor)"
	@echo ""

# 服务命令
build: ## 构建镜像
	@echo "$(GREEN)构建镜像...$(NC)"
	docker-compose build --no-cache

up: ## 启动服务
	@echo "$(GREEN)启动服务...$(NC)"
	@echo "$(YELLOW)确保基础设施服务已启动...$(NC)"
	@cd ../infra && docker-compose up -d
	docker-compose up -d
	@echo "$(GREEN)服务已启动: $(API_BASE_URL)$(NC)"

down: ## 停止服务
	@echo "$(GREEN)停止服务...$(NC)"
	docker-compose down

restart: ## 重启服务
	@echo "$(GREEN)重启服务...$(NC)"
	docker-compose restart

logs: ## 查看服务日志
	@echo "$(GREEN)查看服务日志...$(NC)"
	docker-compose logs -f verification-service

status: ## 查看服务状态
	@echo "$(GREEN)服务状态:$(NC)"
	docker-compose ps

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

# 部署命令
deploy: ## 部署到生产环境
	@echo "$(GREEN)部署到生产环境...$(NC)"
	@echo "$(YELLOW)请确保已配置生产环境变量$(NC)"
	docker-compose -f docker-compose.prod.yml up -d

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

# 日志命令
logs-error: ## 查看错误日志
	@echo "$(GREEN)查看错误日志...$(NC)"
	docker-compose logs | grep -i error

# 环境命令
env-check: ## 检查环境变量
	@echo "$(GREEN)检查环境变量...$(NC)"
	@echo "$(YELLOW)必需的环境变量:$(NC)"
	@echo "  BREVO_API_KEY: $(if $(BREVO_API_KEY),$(GREEN)已设置$(NC),$(RED)未设置$(NC))"
	@echo "  BREVO_FROM_EMAIL: $(if $(BREVO_FROM_EMAIL),$(GREEN)已设置$(NC),$(RED)未设置$(NC))"
	@echo "  BREVO_FROM_NAME: $(if $(BREVO_FROM_NAME),$(GREEN)已设置$(NC),$(RED)未设置$(NC))"
	@echo "  DATABASE_URL: $(if $(DATABASE_URL),$(GREEN)已设置$(NC),$(RED)未设置$(NC))"
	@echo "  REDIS_URL: $(if $(REDIS_URL),$(GREEN)已设置$(NC),$(RED)未设置$(NC))"

# 快速启动命令
quick-start: ## 快速启动（启动基础设施 + 服务）
	@echo "$(GREEN)快速启动 Verification API 服务...$(NC)"
	@make up
	@echo "$(GREEN)等待服务启动...$(NC)"
	@sleep 10
	@make test-health
	@echo "$(GREEN)服务启动完成！$(NC)"
	@echo "$(YELLOW)访问地址: $(API_BASE_URL)$(NC)"

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
