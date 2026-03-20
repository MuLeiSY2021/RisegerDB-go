package function

import (
	"math"
	"strconv"

	"github.com/riseger/riseger-go/internal/compile/parser"
	"github.com/riseger/riseger-go/pkg/rtree"
)

// ExtractScopes walks the WHERE AST and extracts spatial rectangles that can
// be used for R-tree index lookups. Returns nil when the WHERE clause cannot
// be bounded (fallback to full scan).
func ExtractScopes(whereNode *parser.Node, threshold float64) []*rtree.Rect {
	if whereNode == nil {
		return nil
	}
	return extractFromNode(whereNode, threshold)
}

func extractFromNode(node *parser.Node, threshold float64) []*rtree.Rect {
	switch node.Type {
	case parser.NodeWhere:
		if len(node.Children) == 0 {
			return nil
		}
		return extractFromNode(node.Children[0], threshold)

	case parser.NodeAnd:
		left := extractFromNode(node.Children[0], threshold)
		right := extractFromNode(node.Children[1], threshold)
		if left == nil {
			return right
		}
		if right == nil {
			return left
		}
		// Cartesian intersection: for each pair, compute intersect; keep non-empty.
		var result []*rtree.Rect
		for _, l := range left {
			for _, r := range right {
				inter := intersectRects(l, r, threshold)
				if inter != nil {
					result = append(result, inter)
				}
			}
		}
		if len(result) == 0 {
			// All pairs are disjoint — impossible condition, but return empty
			// slice (not nil) so the caller knows to return zero results.
			return []*rtree.Rect{}
		}
		return result

	case parser.NodeOr:
		left := extractFromNode(node.Children[0], threshold)
		right := extractFromNode(node.Children[1], threshold)
		if left == nil || right == nil {
			return nil // one side unbounded → full scan
		}
		return append(left, right...)

	case parser.NodeIn:
		rect := evalStaticRect(node.Children[0], threshold)
		if rect == nil {
			return nil
		}
		return []*rtree.Rect{expandScope(rect, threshold)}

	case parser.NodeNot, parser.NodeOut:
		return nil // complement cannot be bounded

	default:
		return nil
	}
}

// evalStaticRect tries to statically evaluate a spatial expression node into
// a concrete Rect. Returns nil if the expression contains dynamic references
// (e.g. attribute lookups).
func evalStaticRect(node *parser.Node, threshold float64) *rtree.Rect {
	switch node.Type {
	case parser.NodeRect:
		// NodeRect: Children[0] = coord, Children[1] = size
		coord := evalStaticCoord(node.Children[0], threshold)
		if coord == nil {
			return nil
		}
		size := evalStaticNumber(node.Children[1])
		if math.IsNaN(size) {
			return nil
		}
		cx := (coord.MinX() + coord.MaxX()) / 2
		cy := (coord.MinY() + coord.MaxY()) / 2
		return rtree.NewRect(cx-size, cy-size, cx+size, cy+size, threshold)

	case parser.NodeCoordToRect:
		return evalStaticCoord(node.Children[0], threshold)

	case parser.NodeCoord:
		return evalStaticCoord(node, threshold)

	default:
		return nil // dynamic expression (attribute reference, etc.)
	}
}

func evalStaticCoord(node *parser.Node, threshold float64) *rtree.Rect {
	if node.Type != parser.NodeCoord {
		return nil
	}
	x := evalStaticNumber(node.Children[0])
	y := evalStaticNumber(node.Children[1])
	if math.IsNaN(x) || math.IsNaN(y) {
		return nil
	}
	return rtree.NewRect(x, y, x, y, threshold)
}

// evalStaticNumber evaluates a numeric expression that contains only literal
// numbers and arithmetic operators. Returns NaN if the expression is dynamic.
func evalStaticNumber(node *parser.Node) float64 {
	switch node.Type {
	case parser.NodeNumber:
		v, err := strconv.ParseFloat(node.Value, 64)
		if err != nil {
			return math.NaN()
		}
		return v

	case parser.NodeNegate:
		v := evalStaticNumber(node.Children[0])
		return -v

	case parser.NodeAdd:
		return evalStaticBinop(node, func(a, b float64) float64 { return a + b })
	case parser.NodeSub:
		return evalStaticBinop(node, func(a, b float64) float64 { return a - b })
	case parser.NodeMul:
		return evalStaticBinop(node, func(a, b float64) float64 { return a * b })
	case parser.NodeDiv:
		return evalStaticBinop(node, func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		})

	default:
		return math.NaN()
	}
}

func evalStaticBinop(node *parser.Node, op func(a, b float64) float64) float64 {
	a := evalStaticNumber(node.Children[0])
	b := evalStaticNumber(node.Children[1])
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.NaN()
	}
	return op(a, b)
}

// intersectRects computes the intersection of two rectangles.
// Returns nil if they do not overlap (empty intersection).
func intersectRects(a, b *rtree.Rect, threshold float64) *rtree.Rect {
	minX := math.Max(a.MinX(), b.MinX())
	minY := math.Max(a.MinY(), b.MinY())
	maxX := math.Min(a.MaxX(), b.MaxX())
	maxY := math.Min(a.MaxY(), b.MaxY())
	if minX > maxX || minY > maxY {
		return nil
	}
	return rtree.NewRect(minX, minY, maxX, maxY, threshold)
}

// expandScope expands each side of the rectangle by threshold so that
// Search (which uses Intersects/Overlap > 0) can find point elements and
// elements touching the boundary.
func expandScope(scope *rtree.Rect, threshold float64) *rtree.Rect {
	if threshold <= 0 {
		threshold = 1.0 // safe default for non-truncated coordinates
	}
	return rtree.NewRect(
		scope.MinX()-threshold,
		scope.MinY()-threshold,
		scope.MaxX()+threshold,
		scope.MaxY()+threshold,
		0, // no truncation on the search scope itself
	)
}
