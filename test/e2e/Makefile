# Print the help menu.
.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: create-cluster
create-cluster: ## Create a kind cluster for converter e2e tests.
	./tools/create-cluster.sh

.PHONY: deploy-controllers
deploy-controllers: ## Deploy nginx ingress controller and Envoy Gateway controller to the kind cluster.
	./tools/nginx-deploy.sh
	./tools/eg-deploy.sh

.PHONY: deploy-e2e-apps
deploy-e2e-apps:  ## Deploy e2e test applications to the kind cluster
	kubectl apply -f ./testdata/apps