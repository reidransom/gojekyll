package filters

import (
	"sync"

	sass "github.com/bep/godartsass/v2"
	"github.com/reidransom/jigyll/internal/sasserrors"
)

// SASS transpiler singleton for the scssify filter. dart-sass is resolved from
// PATH (sass.Options{}); see renderers.getSassTranspiler for the build-side copy.
var (
	scssifyTranspiler     *sass.Transpiler
	scssifyTranspilerErr  error
	scssifyTranspilerOnce sync.Once
)

func getScssifyTranspiler() (*sass.Transpiler, error) {
	scssifyTranspilerOnce.Do(func() {
		scssifyTranspiler, scssifyTranspilerErr = sass.Start(sass.Options{})
		scssifyTranspilerErr = sasserrors.Enhance(scssifyTranspilerErr)
	})
	return scssifyTranspiler, scssifyTranspilerErr
}

func sassifyFilter(s string) (string, error) {
	comp, err := getScssifyTranspiler()
	if err != nil {
		return "", err
	}
	res, err := comp.Execute(sass.Args{
		Source:       s,
		SourceSyntax: sass.SourceSyntaxSASS,
	})
	return res.CSS, err
}

func scssifyFilter(s string, includePaths []string) (string, error) {
	comp, err := getScssifyTranspiler()
	if err != nil {
		return "", err
	}
	res, err := comp.Execute(sass.Args{
		Source:       s,
		IncludePaths: includePaths,
	})
	return res.CSS, err
}
