Converts all run configurations from GoLand to VSCode launch.json format. WILL OVERWRITE ANY EXISTING launch.json

Install:

```
go install cmd/gotakeoff/gotakeoff.go
```

Usage:

```
gotakeoff <path_to_GoLand_project>
```

then `.vscode` directory with `launch.json` will be created with all migrated run configurations, should work out of the box