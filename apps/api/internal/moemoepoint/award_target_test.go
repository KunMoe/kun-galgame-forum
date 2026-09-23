package moemoepoint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// ReasonLiked means "someone liked your content", so the delta belongs to the
// author. Galgame-resource likes credited the liker instead for as long as the
// feature existed — 8,510 like rows on production paid the wrong person, and the
// matching notification went to the author, so nothing in the product disagreed
// with itself loudly enough to be noticed (fixed 2026-09-13, PR #170).
//
// Every like path passes the author down as a field of the row it just read
// (row.UserID, topic.UserID, ownerID); the actor arrives as the handler's own
// parameter. That difference is the whole bug, and it is visible in the syntax.
var actorParams = map[string]bool{
	"userID":        true,
	"uid":           true,
	"actorID":       true,
	"likerID":       true,
	"senderID":      true,
	"currentUserID": true,
}

// The award shapes in this repo: the interaction helpers take the transaction
// first, the package functions do not, and the v1 wall service calls an
// injected award func with the package function's arguments.
var awardShapes = map[string]struct{ target, reason int }{
	"AdjustMoemoepoint": {1, 3},
	"adjustMoemoepoint": {1, 3},
	"Award":             {0, 2},
	"AwardSync":         {0, 2},
	"award":             {0, 2},
}

// v1 collects its awards as struct literals and flushes them after the commit,
// so the target is a field and not an argument. Deleting the legacy routes on
// 2026-09-22 took every call-shaped liked award on the topic side with them:
// the walk below still passed while covering none of the paths that now serve
// the site.
const awardLiteral = "pendingAward"

func TestLikeAwardsCreditTheAuthorNotTheActor(t *testing.T) {
	found := 0
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if lit, ok := n.(*ast.CompositeLit); ok {
				if id, ok := lit.Type.(*ast.Ident); ok && id.Name == awardLiteral {
					fields := map[string]ast.Expr{}
					for _, el := range lit.Elts {
						kv, ok := el.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						if key, ok := kv.Key.(*ast.Ident); ok {
							fields[key.Name] = kv.Value
						}
					}
					if reason, ok := fields["reason"]; ok && mentions(reason, "ReasonLiked") {
						found++
						if id, ok := fields["userID"].(*ast.Ident); ok && actorParams[id.Name] {
							t.Errorf("%s: a liked award credits %q — that is whoever pressed the "+
								"button, not the author of what they liked",
								fset.Position(lit.Pos()), id.Name)
						}
					}
				}
				return true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			shape, ok := awardShapes[calleeName(call.Fun)]
			if !ok || len(call.Args) <= shape.reason || len(call.Args) <= shape.target {
				return true
			}
			if !mentions(call.Args[shape.reason], "ReasonLiked") {
				return true
			}
			found++
			if id, ok := call.Args[shape.target].(*ast.Ident); ok && actorParams[id.Name] {
				t.Errorf("%s: a liked award credits %q — that is whoever pressed the "+
					"button, not the author of what they liked",
					fset.Position(call.Pos()), id.Name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk the service tree: %v", err)
	}

	// Without this the check passes an empty room: rename the helper, move the
	// reason behind a variable, and every assertion above simply stops running.
	if found < 15 {
		t.Fatalf("only %d liked-award sites seen, expected at least 15 — this check "+
			"has gone blind rather than the awards having gone away", found)
	}
}

func calleeName(fun ast.Expr) string {
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		return f.Sel.Name
	case *ast.Ident:
		return f.Name
	}
	return ""
}

func mentions(expr ast.Expr, name string) bool {
	hit := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == name {
			hit = true
		}
		return !hit
	})
	return hit
}
