## Usage

Run e2e tests:

    dagger call --source=".:default" etoe --db-mode=USE_CACHED

Generate new database dumps:

    dagger call --source=".:default" generate-dumps export --path=./hack/ccp/integration/data/

Use of `dagger call --progress=plain` yields clearer output during debugging.
