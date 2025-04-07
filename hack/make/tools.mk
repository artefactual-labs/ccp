# List of tools.
TOOLS = \
    buf \
    go-enum \
    golangci-lint \
    gotestsum \
    mockgen \
    sqlc \
    tparse \
    go-mod-outdated

# Pattern rule to install each tool.
tool-%:
	@go tool bine get $* 1> /dev/null
