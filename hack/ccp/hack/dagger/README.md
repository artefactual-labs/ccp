## Usage

### Run e2e tests

    dagger call --source=".:default" etoe --db-mode=USE_DUMPS

The `USE_DUMPS` mode (default) uses pre-built SQL dumps to speed up migrations.
Other modes available are: `USE_CACHED` (rely on existing state) and
`FORCE_DROP` (drops existing dbs if exist).

### Genereate SQL dumps

The e2e tests load existing SQL dumps by default, but you can update them with:

    dagger call --source=".:default" generate-dumps export --path=hack/ccp/e2e/testdata/dumps

### Troubleshooting

Use of `dagger call --progress=plain` yields clearer output during debugging.
