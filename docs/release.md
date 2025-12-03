# Release

To create a new release

1. Update the chart and appVersion in [Chart.yaml](../chart/Chart.yaml)
2. Merge the PR
3. Push the new tag with e.g.

```bash
git switch main
git pull
git tag v<x>.<y>.<z>
git push origin tag v<x>.<y>.<z>
```
