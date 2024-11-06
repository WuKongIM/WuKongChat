build:
	docker build -t wukongimbusinessextra .
push:
	docker tag wukongimbusinessextra registry.cn-shanghai.aliyuncs.com/wukongim/wukongimbusinessextra:latest
	docker push registry.cn-shanghai.aliyuncs.com/wukongim/wukongchatserver:latest
deploy:
	docker build -t wukongimbusinessextra .
	docker tag wukongimbusinessextra registry.cn-shanghai.aliyuncs.com/wukongim/wukongimbusinessextra:latest
	docker push registry.cn-shanghai.aliyuncs.com/wukongim/wukongimbusinessextra:latest
deploy-v1.0:
	docker build -t wukongimbusinessextra .
	docker tag wukongimbusinessextra registry.cn-shanghai.aliyuncs.com/wukongim/wukongimbusinessextra:v1.0
	docker push registry.cn-shanghai.aliyuncs.com/wukongim/wukongimbusinessextra:v1.0
run-dev:
	docker-compose build;docker-compose up -d
stop-dev:
	docker-compose stop
env-test:
	docker-compose -f ./testenv/docker-compose.yaml up -d 