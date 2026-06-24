setup:
	bash infra/scripts/setup.sh

deploy:
	bash infra/scripts/deploy.sh dev
	bash infra/scripts/deploy-apps.sh

mesh:
	bash infra/scripts/istio-install.sh
	kubectl apply -f infra/istio/

mesh-down:
	kubectl delete -f infra/istio/ --ignore-not-found || true
	bash infra/scripts/istio-uninstall.sh

reset:
	bash infra/scripts/teardown.sh