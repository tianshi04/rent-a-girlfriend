setup:
	bash infra/scripts/registry.sh
	bash infra/scripts/setup.sh

deploy:
	kubectl apply -f infra/k8s/base/

reset:
	bash infra/scripts/teardown.sh