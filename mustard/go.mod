module github.com/danfragoso/thdwb/mustard

go 1.16

require (
	github.com/danfragoso/thdwb/assets v0.0.0-20210411220950-eaec6f13eccb
	github.com/danfragoso/thdwb/gg v0.0.0-20210411220950-eaec6f13eccb
	github.com/go-gl/gl v0.0.0-20210501111010-69f74958bac0
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20210410170116-ea3d685f79fb
	github.com/goki/freetype v0.0.0-20181231101311-fa8a33aabaff
)

replace (
	github.com/danfragoso/thdwb/assets => ../assets
	github.com/danfragoso/thdwb/gg => ../gg
)
