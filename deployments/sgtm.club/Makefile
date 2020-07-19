.PHONY: all
all: up logs

.PHONY: up
up:
	docker-compose up -d

.PHONY: logs
logs:
	while true; do make logs-once; sleep 1; done

.PHONY: logs-once
logs-once:
	docker-compose logs --tail=1000 -f

.PHONY: ps pull restart stop kill
ps pull restart stop kill:
	docker-compose $@

.PHONY: dbdump
dbdump:
	sqlite3 sgtm.db .dump

.PHONY: dbflush
dbflush:
	rm -f sgtm.db

.PHONY: down
down:
	docker-compose down --remove-orphans

.PHONY: re
re: pull up

.PHONY: backup
backup:
	sqlite3 sgtm.db -cmd '.timeout 20000' ".backup backups/`date +%Y-%m-%d_%H:%M:%S`.sqlite3"
