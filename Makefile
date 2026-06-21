PYTHON ?= python -X utf8

.PHONY: setup deploy mesh mesh-down reset seed-db test-e2e test-e2e-system test-flow1 test-flow2 test-flow3 test-flow4 test-flow5 test-flow6 test-flow7 test-e2e-deps test-e2e-create-user

setup:
	bash infra/scripts/registry.sh
	bash infra/scripts/setup.sh

deploy:
	bash infra/scripts/deploy.sh dev

mesh:
	bash infra/scripts/istio-install.sh
	kubectl apply -f infra/istio/

mesh-down:
	kubectl delete -f infra/istio/ --ignore-not-found || true
	bash infra/scripts/istio-uninstall.sh

reset:
	bash infra/scripts/teardown.sh

seed-db:
	docker compose run --rm db-seeder

test-e2e-deps:
	$(PYTHON) -m pip install requests

test-e2e:
	$(PYTHON) tests/e2e/run_all_tests.py

test: test-e2e

test-e2e-system:
	$(PYTHON) tests/e2e/test_system_flow.py

test-flow1:
	$(PYTHON) tests/e2e/test_flow_1_auth_upgrade.py

test-flow2:
	$(PYTHON) tests/e2e/test_flow_2_companions.py

test-flow3:
	$(PYTHON) tests/e2e/test_flow_3_topup.py

test-flow4:
	$(PYTHON) tests/e2e/test_flow_4_booking_saga.py

test-flow5:
	$(PYTHON) tests/e2e/test_flow_5_disputes.py

test-flow6:
	$(PYTHON) tests/e2e/test_flow_6_resilience_idempotency.py

test-flow7:
	$(PYTHON) tests/e2e/test_flow_7_error_cases.py

test-e2e-create-user:
	$(PYTHON) tests/e2e/test_auth_helper.py