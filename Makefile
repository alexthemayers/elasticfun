
up:
	docker compose up -d

down:
	docker compose down receiver caller middleman

restart: down
	docker image rm elasticfun-receiver elasticfun-caller elasticfun-middleman
	docker compose up -d
