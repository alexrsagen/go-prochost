VERBOSE_1 := -v
VERBOSE_2 := -v -x

WHAT := prochost

build:
	@for target in $(WHAT); do \
		$(BUILD_ENV_FLAGS) go build $(VERBOSE_$(V)) -o ./bin/$$target ./cmd/$$target; \
	done

clean:
	rm -rf ./bin/*

help:
	@echo "Influential make variables"
	@echo "  V                 - Build verbosity {0,1,2}."
	@echo "  BUILD_ENV_FLAGS   - Environment added to 'go build'."
	@echo "  WHAT              - Command to build. (e.g. WHAT=prochost)"
