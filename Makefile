setup:
	bash infra/scripts/01-create-cluster.sh
	bash infra/scripts/02-install-mesh.sh
	kubectl apply -f infra/istio/
	bash infra/scripts/03-deploy-infra.sh dev
	bash infra/scripts/04-deploy-apps.sh

reset:
	bash infra/scripts/destroy-cluster.sh