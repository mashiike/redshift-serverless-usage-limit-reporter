export AWS_ACCOUNT_ID := $(shell aws sts get-caller-identity --query 'Account' --output text)
.PHONY: clean deploy-all

bootstrap: ../*.go ../*.go ../go.*
	GOARCH=arm64 GOOS=linux go build -o bootstrap ../*.go

clean:
	rm -f bootstrap
	rm -f *_dummy.zip

lambroll/deploy: bootstrap
	lambroll --log-level debug deploy
	$(MAKE) clean

lambroll/logs:
	lambroll logs --follow --format=short

.PHONY: terraform/init
terraform/init:
	terraform providers lock \
		-platform=darwin_amd64 \
		-platform=darwin_arm64 \
		-platform=linux_amd64 \
		-platform=linux_arm64 \
		-platform=windows_amd64
	terraform init


.PHONY: terraform/plan
terraform/plan:
	terraform plan


.PHONY: terraform/apply
terraform/apply:
	terraform apply
	$(MAKE) clean

.PHONY: terraform/destroy
terraform/destroy:
	terraform destroy
