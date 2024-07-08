$(call _assert_var,MAKEDIR)
$(call _conditional_include,$(MAKEDIR)/base.mk)
$(call _assert_var,CACHE_VERSIONS)
$(call _assert_var,CACHE_BIN)

OAPI_CODEGEN_VERSION ?= 2.3.0

OAPI_CODEGEN := $(CACHE_VERSIONS)/oapi-codegen/$(OAPI_CODEGEN_VERSION)
$(OAPI_CODEGEN):
	rm -f $(CACHE_BIN)/oapi-codegen
	mkdir -p $(CACHE_BIN)
	env GOBIN=$(CACHE_BIN) go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v$(OAPI_CODEGEN_VERSION)
	chmod +x $(CACHE_BIN)/oapi-codegen
	rm -rf $(dir $(OAPI_CODEGEN))
	mkdir -p $(dir $(OAPI_CODEGEN))
	touch $(OAPI_CODEGEN)
