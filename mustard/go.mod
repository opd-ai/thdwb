module github.com/danfragoso/thdwb/mustard

go 1.16

require (
	github.com/danfragoso/thdwb/assets v0.0.0-20210411220950-eaec6f13eccb
	github.com/danfragoso/thdwb/gg v0.0.0-20210411220950-eaec6f13eccb
	github.com/go-gl/gl v0.0.0-20260331235117-4566fea9a276
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20260802143932-8fa725040a18
	github.com/goki/freetype v1.0.5
)

replace (
	github.com/danfragoso/thdwb/assets => ../assets
	github.com/danfragoso/thdwb/gg => ../gg
)
