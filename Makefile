dev:
	go run main.go

build:
	gcloud builds submit --tag gcr.io/beatbrain-dev/melodex

deploy:
	gcloud run deploy melodex \
	--image gcr.io/beatbrain-dev/melodex \
	--platform managed

ship:
	make build
	make deploy