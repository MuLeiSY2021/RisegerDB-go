package function

import (
	"testing"

	"github.com/riseger/riseger-go/internal/compile/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseWhere(t *testing.T, sql string) *parser.Node {
	t.Helper()
	// Wrap in a minimal valid query so the parser produces a WHERE node.
	full := "SEARCH name WHERE " + sql
	ast, err := parser.Parse(full)
	require.NoError(t, err, "parse %q", full)
	// AST: NodeSearch → Children[0]=NodeStrings, Children[1]=NodeWhere
	require.GreaterOrEqual(t, len(ast.Children), 2, "expected WHERE child")
	return ast.Children[1]
}

func TestExtractScopes_SingleRect(t *testing.T) {
	where := parseWhere(t, "IN RECT([10, 20], 50)")
	scopes := ExtractScopes(where, 0.5)
	require.Len(t, scopes, 1)
	r := scopes[0]
	// RECT([10,20], 50) → center (10,20), half-size 50 → [-40, -30, 60, 70]
	// expanded by threshold 0.5 → [-40.5, -30.5, 60.5, 70.5]
	assert.InDelta(t, -40.5, r.MinX(), 0.01)
	assert.InDelta(t, -30.5, r.MinY(), 0.01)
	assert.InDelta(t, 60.5, r.MaxX(), 0.01)
	assert.InDelta(t, 70.5, r.MaxY(), 0.01)
}

func TestExtractScopes_InRectAndComparison(t *testing.T) {
	where := parseWhere(t, "IN RECT([10, 20], 50) AND area > 100")
	scopes := ExtractScopes(where, 0.5)
	require.Len(t, scopes, 1, "comparison branch ignored, single rect extracted")
}

func TestExtractScopes_TwoRectsAnd(t *testing.T) {
	// IN RECT([0,0], 100) AND IN RECT([50,50], 100)
	// Rect A: [-100, -100, 100, 100] expanded → [-100.5, -100.5, 100.5, 100.5]
	// Rect B: [-50, -50, 150, 150] expanded → [-50.5, -50.5, 150.5, 150.5]
	// Intersection: [-50.5, -50.5, 100.5, 100.5]
	where := parseWhere(t, "IN RECT([0, 0], 100) AND IN RECT([50, 50], 100)")
	scopes := ExtractScopes(where, 0.5)
	require.Len(t, scopes, 1)
	r := scopes[0]
	assert.InDelta(t, -50.5, r.MinX(), 0.01)
	assert.InDelta(t, -50.5, r.MinY(), 0.01)
	assert.InDelta(t, 100.5, r.MaxX(), 0.01)
	assert.InDelta(t, 100.5, r.MaxY(), 0.01)
}

func TestExtractScopes_TwoRectsOr(t *testing.T) {
	where := parseWhere(t, "IN RECT([0, 0], 10) OR IN RECT([100, 100], 10)")
	scopes := ExtractScopes(where, 0.5)
	require.Len(t, scopes, 2, "OR produces two scopes")
}

func TestExtractScopes_OrWithNonSpatial(t *testing.T) {
	where := parseWhere(t, "IN RECT([0, 0], 10) OR area > 100")
	scopes := ExtractScopes(where, 0.5)
	assert.Nil(t, scopes, "one unbounded OR branch → nil (full scan)")
}

func TestExtractScopes_NotInRect(t *testing.T) {
	where := parseWhere(t, "NOT IN RECT([0, 0], 10)")
	scopes := ExtractScopes(where, 0.5)
	assert.Nil(t, scopes, "NOT IN → nil (complement)")
}

func TestExtractScopes_PureComparison(t *testing.T) {
	where := parseWhere(t, "area > 100")
	scopes := ExtractScopes(where, 0.5)
	assert.Nil(t, scopes, "no spatial predicate → nil")
}

func TestExtractScopes_DisjointAndRects(t *testing.T) {
	// Two non-overlapping rects with AND → empty intersection
	where := parseWhere(t, "IN RECT([0, 0], 1) AND IN RECT([1000, 1000], 1)")
	scopes := ExtractScopes(where, 0.5)
	require.NotNil(t, scopes, "should be non-nil empty slice")
	assert.Len(t, scopes, 0, "disjoint AND → empty scopes (zero results)")
}

func TestExtractScopes_CoordToRect(t *testing.T) {
	// IN [10, 20] → NodeIn → NodeCoordToRect → NodeCoord
	// Point rect [10, 20, 10, 20], expanded by threshold=0.5 → [9.5, 19.5, 10.5, 20.5]
	where := parseWhere(t, "IN [10, 20]")
	scopes := ExtractScopes(where, 0.5)
	require.Len(t, scopes, 1)
	r := scopes[0]
	assert.InDelta(t, 9.5, r.MinX(), 0.01)
	assert.InDelta(t, 19.5, r.MinY(), 0.01)
	assert.InDelta(t, 10.5, r.MaxX(), 0.01)
	assert.InDelta(t, 20.5, r.MaxY(), 0.01)
}

func TestExtractScopes_NilNode(t *testing.T) {
	scopes := ExtractScopes(nil, 0.5)
	assert.Nil(t, scopes)
}

func TestExtractScopes_ComplexOrAnd(t *testing.T) {
	// (IN A AND x > 50) OR (IN B AND y > 100)
	// Left branch: IN A (AND with comparison → just A)
	// Right branch: IN B (AND with comparison → just B)
	// OR: [A, B]
	where := parseWhere(t, "(IN RECT([0, 0], 10) AND area > 50) OR (IN RECT([100, 100], 10) AND area > 100)")
	scopes := ExtractScopes(where, 0.5)
	require.Len(t, scopes, 2)
}
