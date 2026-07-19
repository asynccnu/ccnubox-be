package errs

import (
	"errors"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"net/http"
	"strings"
	"testing"

	b_errorx "github.com/asynccnu/ccnubox-be/bff/pkg/errorx"
)

func TestErrorCodesAreUniqueAndWellFormed(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "codes.go", nil, 0)
	if err != nil {
		t.Fatalf("parse codes.go: %v", err)
	}

	pkg, err := new(types.Config).Check("errs", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("type-check codes.go: %v", err)
	}

	compatibilityAliases := map[string]struct{}{
		"ADD_CLASS_CONFLICT_ERROR_CODE":   {},
		"UNAUTHORIED_ERROR_CODE":          {},
		"USER_SID_OR_PASSPORD_ERROR_CODE": {},
	}
	seen := make(map[int64]string)
	for _, name := range pkg.Scope().Names() {
		if !strings.HasSuffix(name, "_ERROR_CODE") {
			continue
		}
		if _, ok := compatibilityAliases[name]; ok {
			continue
		}

		obj, ok := pkg.Scope().Lookup(name).(*types.Const)
		if !ok {
			t.Fatalf("%s is not a constant", name)
		}
		code, ok := constant.Int64Val(obj.Val())
		if !ok {
			t.Fatalf("%s is not an integer", name)
		}
		if code < 40000 || code > 59999 || (code/10000 != 4 && code/10000 != 5) {
			t.Errorf("%s=%d is not a five-digit client/server error code", name, code)
		}
		if previous, exists := seen[code]; exists {
			t.Errorf("%s and %s both use error code %d", previous, name, code)
		}
		seen[code] = name
	}
}

func TestEveryErrorFactoryUsesItsOwnCode(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "errs.go", nil, 0)
	if err != nil {
		t.Fatalf("parse errs.go: %v", err)
	}

	ast.Inspect(file, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok || len(spec.Names) != 1 || len(spec.Values) != 1 {
			return true
		}

		name := spec.Names[0].Name
		outerCall, ok := spec.Values[0].(*ast.CallExpr)
		if !ok || !isSelector(outerCall.Fun, "errorx", "FormatErrorFunc") {
			return true // Compatibility aliases are identifiers, not factories.
		}
		if len(outerCall.Args) != 1 {
			t.Errorf("%s must wrap exactly one error definition", name)
			return true
		}

		constructor, ok := outerCall.Args[0].(*ast.CallExpr)
		if !ok || !isSelector(constructor.Fun, "b_errorx", "New") || len(constructor.Args) != 3 {
			t.Errorf("%s does not use b_errorx.New(httpStatus, code, message)", name)
			return true
		}
		code, ok := constructor.Args[1].(*ast.Ident)
		if !ok || code.Name != name+"_CODE" {
			got := "non-identifier"
			if ok {
				got = code.Name
			}
			t.Errorf("%s uses %s; want %s_CODE", name, got, name)
		}
		return true
	})
}

func isSelector(expr ast.Expr, packageName, selectorName string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != selectorName {
		return false
	}
	identifier, ok := selector.X.(*ast.Ident)
	return ok && identifier.Name == packageName
}

func TestRepresentativeErrorMappings(t *testing.T) {
	cause := errors.New("cause")
	tests := []struct {
		name       string
		build      func(error) error
		httpStatus int
		code       int
	}{
		{name: "common client", build: BAD_ENTITY_ERROR, httpStatus: http.StatusUnprocessableEntity, code: BAD_ENTITY_ERROR_CODE},
		{name: "module server", build: GET_BANNER_ERROR, httpStatus: http.StatusInternalServerError, code: GET_BANNER_ERROR_CODE},
		{name: "class conflict", build: CLASS_SCHEDULE_CONFLICT_ERROR, httpStatus: http.StatusConflict, code: CLASS_SCHEDULE_CONFLICT_ERROR_CODE},
		{name: "class exists", build: CLASS_ALREADY_EXISTS_ERROR, httpStatus: http.StatusConflict, code: CLASS_ALREADY_EXISTS_ERROR_CODE},
		{name: "expired authorization", build: AUTH_EXPIRED_ERROR, httpStatus: http.StatusUnauthorized, code: AUTH_EXPIRED_ERROR_CODE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got *b_errorx.CustomError
			if !errors.As(tt.build(cause), &got) {
				t.Fatal("wrapped error does not contain a CustomError")
			}
			if got.HttpCode != tt.httpStatus {
				t.Errorf("HTTP status = %d, want %d", got.HttpCode, tt.httpStatus)
			}
			if got.Code != tt.code {
				t.Errorf("error code = %d, want %d", got.Code, tt.code)
			}
		})
	}
}
