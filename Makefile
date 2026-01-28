image:
	docker buildx build --platform linux/amd64,linux/arm64 -t public.ecr.aws/axatol/pointsman:latest .

build:
	go build -o pointsman

run: build
	./pointsman
