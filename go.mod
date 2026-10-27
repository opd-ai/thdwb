module github.com/danfragoso/thdwb

go 1.26.0

replace (
	github.com/danfragoso/thdwb/assets => ./assets
	github.com/danfragoso/thdwb/bun => ./bun
	github.com/danfragoso/thdwb/gg => ./gg
	github.com/danfragoso/thdwb/hotdog => ./hotdog
	github.com/danfragoso/thdwb/ketchup => ./ketchup
	github.com/danfragoso/thdwb/mayo => ./mayo
	github.com/danfragoso/thdwb/mustard => ./mustard
	github.com/danfragoso/thdwb/pages => ./pages
	github.com/danfragoso/thdwb/profiler => ./profiler
	github.com/danfragoso/thdwb/sauce => ./sauce
)

require (
	github.com/andybalholm/cascadia v1.3.5
	github.com/dop251/goja v0.0.0-20261026132650-6b6a90e51946
	github.com/goki/freetype v1.0.5
	github.com/hajimehoshi/ebiten/v2 v2.10.5
	github.com/kjk/flex v0.0.0-20171203210503-ed34d6b6a425
	github.com/stretchr/testify v1.12.1
	golang.org/x/image v0.45.0
	golang.org/x/net v0.61.0
)

require (
	github.com/dlclark/regexp2/v2 v2.8.1 // indirect
	github.com/ebitengine/gomobile v0.0.0-20260820040257-d11f821a26a6 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/purego v0.11.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)
