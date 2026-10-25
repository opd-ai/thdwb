module github.com/danfragoso/thdwb/mustard

go 1.25.0

require (
	github.com/danfragoso/thdwb/assets v0.0.0-20210411220950-eaec6f13eccb
	github.com/danfragoso/thdwb/gg v0.0.0-20210411220950-eaec6f13eccb
	github.com/hajimehoshi/ebiten/v2 v2.10.5
	golang.org/x/image v0.45.0
)

require (
	github.com/ebitengine/gomobile v0.0.0-20260820040257-d11f821a26a6 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/purego v0.11.0 // indirect
	github.com/goki/freetype v1.0.5 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

replace (
	github.com/danfragoso/thdwb/assets => ../assets
	github.com/danfragoso/thdwb/gg => ../gg
)
