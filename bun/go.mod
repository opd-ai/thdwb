module github.com/danfragoso/thdwb/bun

go 1.25.0

require (
	github.com/danfragoso/thdwb/assets v0.0.0-20210411220950-eaec6f13eccb
	github.com/danfragoso/thdwb/gg v0.0.0-20210612223625-beb2b4a85bbb
	github.com/danfragoso/thdwb/hotdog v0.0.0-20210411220950-eaec6f13eccb
	github.com/danfragoso/thdwb/ketchup v0.0.0-20210612223625-beb2b4a85bbb
	github.com/danfragoso/thdwb/sauce v0.0.0-20210411220950-eaec6f13eccb
	github.com/goki/freetype v1.0.5
)

require (
	github.com/andybalholm/cascadia v1.3.4 // indirect
	github.com/danfragoso/thdwb/mayo v0.0.0-20210612223625-beb2b4a85bbb // indirect
	github.com/danfragoso/thdwb/mustard v0.0.0-20210612223625-beb2b4a85bbb // indirect
	github.com/danfragoso/thdwb/pages v0.0.0-20210411220950-eaec6f13eccb // indirect
	github.com/danfragoso/thdwb/profiler v0.0.0-20210612223625-beb2b4a85bbb // indirect
	github.com/go-gl/gl v0.0.0-20260331235117-4566fea9a276 // indirect
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20260802143932-8fa725040a18 // indirect
	golang.org/x/image v0.15.0 // indirect
	golang.org/x/net v0.57.0 // indirect
)

replace (
	github.com/danfragoso/thdwb/assets => ../assets
	github.com/danfragoso/thdwb/gg => ../gg
	github.com/danfragoso/thdwb/hotdog => ../hotdog
	github.com/danfragoso/thdwb/sauce => ../sauce
)
