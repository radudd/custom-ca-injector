IMAGE_REPO ?= quay.io/radudd/custom-ca-injector
IMAGE_NAME ?= custom-ca-injector
IMAGE_TAG  ?= $$(git log --abbrev-commit --format=%h -s | head -n 1)

build:
	echo "Building app"
	go build -v -o ${IMAGE_NAME} ./main.go
    
test:
	echo "Running the tests for $(IMAGE_NAME)..."
	go test -v

image: build-image push-image

build-image: build
	echo "Building the docker image: $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)..."
	docker build -t $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) -f Dockerfile .

push-image: build-image
	echo "Pushing the docker image for $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) and $(IMAGE_REPO)/$(IMAGE_NAME):latest..."
	docker tag $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_REPO)/$(IMAGE_NAME):latest
	docker push $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(IMAGE_REPO)/$(IMAGE_NAME):latest
