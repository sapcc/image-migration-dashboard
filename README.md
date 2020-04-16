# image-migration-dashboard

Dashboard for visualising the ongoing image migration from Quay to
[Keppel](https://github.com/sapcc/keppel).

## Usage

Build with `make`, install with `make install` or `docker build`.

If running inside a cluster:

```
image-migration-dashboard --in-cluster
```

If running outside a cluster:

```
image-migration-dashboard
```

For more info: `image-migration-dashboard --help`.

Dashboard will run at `localhost:80`.
