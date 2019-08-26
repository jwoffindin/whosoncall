
ifndef ASSUMED_ROLE
$(error ASSUMED_ROLE is not set; you probably need to do an assume-role first)
endif

all: build

clean:
	rm -rf bin .serverless

build: clean bin/whos-on-call

test: build
	SLS_DEBUG=\* sls invoke local -f whosoncall

package: build serverless.yml
	sls package

deploy: build serverless.yml
	sls deploy

deploy_production: build serverless.yml
	@echo $(ASSUMED_ROLE)
ifeq ($(ASSUMED_ROLE),orionhealth-saas-mgmt)
	sls deploy --stage prod
else
	$(error Incorrect role $(ASSUMED_ROLE) assumed for prod deployment)
endif

bin/whos-on-call: main.go
	env GOOS=linux go build -mod=vendor -ldflags="-s -w" -o $@ *.go
