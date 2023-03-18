dev:
	air

lint:
	revive -config revive.toml -formatter friendly ./…

swagger:
	swag init --dir ./,./handlers
