setup:
	bash infra/scripts/registry.sh
	bash infra/scripts/setup.sh

deploy:
	kubectl apply -f infra/k8s/base/

mesh:
	bash infra/scripts/istio-install.sh
	kubectl apply -f infra/istio/

mesh-down:
	kubectl delete -f infra/istio/ --ignore-not-found || true
	bash infra/scripts/istio-uninstall.sh

reset:
	bash infra/scripts/teardown.sh