package bun

import (
	"strconv"
	"strings"

	"github.com/danfragoso/thdwb/gg"
	hotdog "github.com/danfragoso/thdwb/hotdog"
	"github.com/kjk/flex"
)

// createFlexNodes creates flex.Nodes for each DOM element and attaches them to the FlexNode field.
// Returns the root flex node.
func createFlexNodes(root *hotdog.NodeDOM) *flex.Node {
	if root.Style.Display == "none" {
		return nil
	}

	// Create flex node for this DOM element
	flexNode := flex.NewNode()
	root.FlexNode = flexNode

	// Apply styles to flex node
	applyStylesToFlexNode(flexNode, root.Style)

	for _, child := range root.Children {
		childFlexNode := createFlexNodes(child)
		if childFlexNode != nil {
			// Add child flex node to parent flex node
			flexNode.InsertChild(childFlexNode, len(flexNode.Children))
		}
	}

	return flexNode
}

// createRenderTree creates a parallel tree of flex.Nodes mirroring the DOM structure.
// Each DOM node gets a corresponding flex.Node stored in its FlexNode field.
func createRenderTree(root *hotdog.NodeDOM) *hotdog.NodeDOM {
	if root.Style.Display == "none" {
		return nil
	}

	node := &hotdog.NodeDOM{
		Style:      root.Style,
		Element:    root.Element,
		Content:    root.Content,
		Attributes: root.Attributes,
	}

	node.RenderBox = &hotdog.RenderBox{}
	node.FlexNode = root.FlexNode // Use the flex node from original DOM

	for _, child := range root.Children {
		r := createRenderTree(child)
		if r != nil {
			r.Parent = node
			node.Children = append(node.Children, r)
		}
	}

	return node
}

func layoutNode(ctx *gg.Context, node *hotdog.NodeDOM) {
	if node.FlexNode == nil {
		return
	}

	flexNode := node.FlexNode.(*flex.Node)

	// Calculate layout for the entire tree from root
	// We only call this on the root node
	if node.Parent == nil {
		flex.CalculateLayout(flexNode, float32(node.RenderBox.Width), float32(node.RenderBox.Height), flex.DirectionLTR)
	}

	// Copy layout results to RenderBox
	copyFlexLayoutToRenderBox(flexNode, node.RenderBox)

	// Recursively process children for painting
	for _, child := range node.Children {
		layoutNode(ctx, child)
	}
}

// copyFlexLayoutToRenderBox copies computed flex layout to hotdog.RenderBox
func copyFlexLayoutToRenderBox(flexNode *flex.Node, renderBox *hotdog.RenderBox) {
	renderBox.Left = float64(flexNode.LayoutGetLeft())
	renderBox.Top = float64(flexNode.LayoutGetTop())
	renderBox.Width = float64(flexNode.LayoutGetWidth())
	renderBox.Height = float64(flexNode.LayoutGetHeight())

	// Also copy margins, padding, borders if needed
	renderBox.MarginTop = float64(flexNode.LayoutGetMargin(flex.EdgeTop))
	renderBox.MarginRight = float64(flexNode.LayoutGetMargin(flex.EdgeRight))
	renderBox.MarginBottom = float64(flexNode.LayoutGetMargin(flex.EdgeBottom))
	renderBox.MarginLeft = float64(flexNode.LayoutGetMargin(flex.EdgeLeft))

	renderBox.PaddingTop = float64(flexNode.LayoutGetPadding(flex.EdgeTop))
	renderBox.PaddingRight = float64(flexNode.LayoutGetPadding(flex.EdgeRight))
	renderBox.PaddingBottom = float64(flexNode.LayoutGetPadding(flex.EdgeBottom))
	renderBox.PaddingLeft = float64(flexNode.LayoutGetPadding(flex.EdgeLeft))
}

func paintText(ctx *gg.Context, node *hotdog.NodeDOM) {
	// Text painting is handled in paintBlockElement/paintInlineElement for text nodes
}

// applyStylesToFlexNode translates hotdog.Stylesheet properties to flex.Node style setters.
func applyStylesToFlexNode(flexNode *flex.Node, style *hotdog.Stylesheet) {
	if style == nil {
		return
	}

	// Display property - only flex and none are supported by flexbox
	switch style.Display {
	case "flex", "inline-flex":
		flexNode.StyleSetDisplay(flex.DisplayFlex)
	case "none":
		flexNode.StyleSetDisplay(flex.DisplayNone)
	default:
		// For block, inline, list-item, etc., we treat as flex with default direction
		// This allows flexbox to handle all layout uniformly
		flexNode.StyleSetDisplay(flex.DisplayFlex)
	}

	// Flex direction
	switch style.FlexDirection {
	case "row":
		flexNode.StyleSetFlexDirection(flex.FlexDirectionRow)
	case "column":
		flexNode.StyleSetFlexDirection(flex.FlexDirectionColumn)
	case "row-reverse":
		flexNode.StyleSetFlexDirection(flex.FlexDirectionRowReverse)
	case "column-reverse":
		flexNode.StyleSetFlexDirection(flex.FlexDirectionColumnReverse)
	}

	// Flex wrap
	switch style.FlexWrap {
	case "wrap":
		flexNode.StyleSetFlexWrap(flex.WrapWrap)
	case "wrap-reverse":
		flexNode.StyleSetFlexWrap(flex.WrapWrapReverse)
	default:
		flexNode.StyleSetFlexWrap(flex.WrapNoWrap)
	}

	// Justify content
	switch style.JustifyContent {
	case "flex-start":
		flexNode.StyleSetJustifyContent(flex.JustifyFlexStart)
	case "flex-end":
		flexNode.StyleSetJustifyContent(flex.JustifyFlexEnd)
	case "center":
		flexNode.StyleSetJustifyContent(flex.JustifyCenter)
	case "space-between":
		flexNode.StyleSetJustifyContent(flex.JustifySpaceBetween)
	case "space-around":
		flexNode.StyleSetJustifyContent(flex.JustifySpaceAround)
	case "space-evenly":
		// Not supported by flex, fall back to space-around
		flexNode.StyleSetJustifyContent(flex.JustifySpaceAround)
	}

	// Align items
	switch style.AlignItems {
	case "flex-start":
		flexNode.StyleSetAlignItems(flex.AlignFlexStart)
	case "flex-end":
		flexNode.StyleSetAlignItems(flex.AlignFlexEnd)
	case "center":
		flexNode.StyleSetAlignItems(flex.AlignCenter)
	case "stretch":
		flexNode.StyleSetAlignItems(flex.AlignStretch)
	case "baseline":
		flexNode.StyleSetAlignItems(flex.AlignBaseline)
	}

	// Align content
	switch style.AlignContent {
	case "flex-start":
		flexNode.StyleSetAlignContent(flex.AlignFlexStart)
	case "flex-end":
		flexNode.StyleSetAlignContent(flex.AlignFlexEnd)
	case "center":
		flexNode.StyleSetAlignContent(flex.AlignCenter)
	case "stretch":
		flexNode.StyleSetAlignContent(flex.AlignStretch)
	case "space-between":
		flexNode.StyleSetAlignContent(flex.AlignSpaceBetween)
	case "space-around":
		flexNode.StyleSetAlignContent(flex.AlignSpaceAround)
	}

	// Align self
	switch style.AlignSelf {
	case "auto":
		flexNode.StyleSetAlignSelf(flex.AlignAuto)
	case "flex-start":
		flexNode.StyleSetAlignSelf(flex.AlignFlexStart)
	case "flex-end":
		flexNode.StyleSetAlignSelf(flex.AlignFlexEnd)
	case "center":
		flexNode.StyleSetAlignSelf(flex.AlignCenter)
	case "stretch":
		flexNode.StyleSetAlignSelf(flex.AlignStretch)
	case "baseline":
		flexNode.StyleSetAlignSelf(flex.AlignBaseline)
	}

	// Flex grow/shrink/basis
	if style.FlexGrow != 0 {
		flexNode.StyleSetFlexGrow(float32(style.FlexGrow))
	}
	if style.FlexShrink != 0 {
		flexNode.StyleSetFlexShrink(float32(style.FlexShrink))
	}
	if style.FlexBasis != "" {
		// Parse flex-basis value
		if style.FlexBasis == "auto" {
			flex.NodeStyleSetFlexBasisAuto(flexNode)
		} else {
			// Try to parse as percentage or float
			val := parseFlexValue(style.FlexBasis)
			if val.Unit == flex.UnitPercent {
				flexNode.StyleSetFlexBasisPercent(val.Value)
			} else {
				flexNode.StyleSetFlexBasis(val.Value)
			}
		}
	}

	// Order - not supported by this flex implementation
	// if style.Order != 0 {
	// 	flexNode.StyleSetOrder(style.Order)
	// }

	// Width/Height
	if style.Width != 0 {
		flexNode.StyleSetWidth(float32(style.Width))
	}
	if style.Height != 0 {
		flexNode.StyleSetHeight(float32(style.Height))
	}
	if style.MinWidth != 0 {
		flexNode.StyleSetMinWidth(float32(style.MinWidth))
	}
	if style.MinHeight != 0 {
		flexNode.StyleSetMinHeight(float32(style.MinHeight))
	}
	if style.MaxWidth != 0 {
		flexNode.StyleSetMaxWidth(float32(style.MaxWidth))
	}
	if style.MaxHeight != 0 {
		flexNode.StyleSetMaxHeight(float32(style.MaxHeight))
	}

	// Position (top, right, bottom, left)
	if style.Position == "absolute" || style.Position == "relative" || style.Position == "fixed" {
		if style.Top != 0 {
			flexNode.StyleSetPosition(flex.EdgeTop, float32(style.Top))
		}
		if style.Right != 0 {
			flexNode.StyleSetPosition(flex.EdgeRight, float32(style.Right))
		}
		if style.Bottom != 0 {
			flexNode.StyleSetPosition(flex.EdgeBottom, float32(style.Bottom))
		}
		if style.Left != 0 {
			flexNode.StyleSetPosition(flex.EdgeLeft, float32(style.Left))
		}
		if style.Position == "absolute" {
			flexNode.StyleSetPositionType(flex.PositionTypeAbsolute)
		} else if style.Position == "relative" {
			flexNode.StyleSetPositionType(flex.PositionTypeRelative)
		}
	}

	// Margins
	if style.MarginTop != 0 {
		flexNode.StyleSetMargin(flex.EdgeTop, float32(style.MarginTop))
	}
	if style.MarginRight != 0 {
		flexNode.StyleSetMargin(flex.EdgeRight, float32(style.MarginRight))
	}
	if style.MarginBottom != 0 {
		flexNode.StyleSetMargin(flex.EdgeBottom, float32(style.MarginBottom))
	}
	if style.MarginLeft != 0 {
		flexNode.StyleSetMargin(flex.EdgeLeft, float32(style.MarginLeft))
	}

	// Padding
	if style.PaddingTop != 0 {
		flexNode.StyleSetPadding(flex.EdgeTop, float32(style.PaddingTop))
	}
	if style.PaddingRight != 0 {
		flexNode.StyleSetPadding(flex.EdgeRight, float32(style.PaddingRight))
	}
	if style.PaddingBottom != 0 {
		flexNode.StyleSetPadding(flex.EdgeBottom, float32(style.PaddingBottom))
	}
	if style.PaddingLeft != 0 {
		flexNode.StyleSetPadding(flex.EdgeLeft, float32(style.PaddingLeft))
	}

	// Borders
	if style.BorderTopWidth != 0 {
		flexNode.StyleSetBorder(flex.EdgeTop, float32(style.BorderTopWidth))
	}
	if style.BorderRightWidth != 0 {
		flexNode.StyleSetBorder(flex.EdgeRight, float32(style.BorderRightWidth))
	}
	if style.BorderBottomWidth != 0 {
		flexNode.StyleSetBorder(flex.EdgeBottom, float32(style.BorderBottomWidth))
	}
	if style.BorderLeftWidth != 0 {
		flexNode.StyleSetBorder(flex.EdgeLeft, float32(style.BorderLeftWidth))
	}

	// Overflow
	switch style.OverflowX {
	case "hidden":
		flexNode.StyleSetOverflow(flex.OverflowHidden)
	case "scroll":
		flexNode.StyleSetOverflow(flex.OverflowScroll)
	default:
		flexNode.StyleSetOverflow(flex.OverflowVisible)
	}
}

// parseFlexValue parses a CSS flex-basis/width/height value string into a flex.Value
func parseFlexValue(s string) flex.Value {
	// Simple parser for values like "100px", "50%", "auto"
	s = strings.TrimSpace(s)
	if s == "auto" {
		return flex.Value{Value: flex.Undefined, Unit: flex.UnitAuto}
	}
	if strings.HasSuffix(s, "%") {
		val := strings.TrimSuffix(s, "%")
		if f, err := strconv.ParseFloat(val, 32); err == nil {
			return flex.Value{Value: float32(f), Unit: flex.UnitPercent}
		}
	}
	// Try to parse as number (assuming px)
	if f, err := strconv.ParseFloat(s, 32); err == nil {
		return flex.Value{Value: float32(f), Unit: flex.UnitPoint}
	}
	// Default
	return flex.Value{Value: flex.Undefined, Unit: flex.UnitAuto}
}
