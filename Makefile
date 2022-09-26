# clean does rm -rf ${DEST} which is too dangerous to let you specify in the environment
DEST = ./dist
DEBUG ?= false
DB_DSN ?= recipes_sqlite.db
DB_DIALECT ?= sqlite3
JWT_SECRET ?= secret
CONFIG ?= gorecipes.conf

.PHONY: test build dist config clean

test:
	go test -v

build: mkdest
	go build -v -o ${DEST}/gorecipes

dist: mkdest config build

config: mkdest
	@echo "Writing Config"
	@touch ${DEST}/${CONFIG}
	@chmod 600 ${DEST}/${CONFIG}
	@echo "{\n\t\"Debug\": ${DEBUG},\n\t\"DbDialect\": \"${DB_DIALECT}\",\n\t\"DbDSN\": \"${DB_DSN}\",\n\t\"JwtSecret\": \"${JWT_SECRET}\",\n\t\"Origins\": \"${ORIGINS}\"\n}" > ${DEST}/${CONFIG}

sqlite: mkdest
	cd bootstrapping && go build -v
	cd bootstrapping && ./bootstrapping
	mv bootstrapping/recipes_sqlite.db ${DEST}
	rm bootstrapping/bootstrapping


mkdest:
	@echo "Making output directory"
	mkdir -p ${DEST}

clean:
	@echo "Cleaning up"
	rm -rf ${DEST}
