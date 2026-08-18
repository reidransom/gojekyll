package tags

import (
	"fmt"
	"path"
	"path/filepath"
	"reflect"

	"github.com/reidransom/liquid/render"
)

type includeRequest struct {
	filename string
	include  map[string]interface{}
}

type resolvedInclude struct {
	filename string
	include  map[string]interface{}
	stack    []includeStackEntry
}

type includeStackEntry struct {
	filename string
	include  map[string]interface{}
}

func (tc tagContext) includeTag(rc render.Context) (string, error) {
	request, err := parseIncludeRequest(rc)
	if err != nil {
		return "", err
	}
	return tc.renderInclude(request, rc)
}

func (tc tagContext) includeRelativeTag(rc render.Context) (string, error) {
	request, err := parseIncludeRequest(rc)
	if err != nil {
		return "", err
	}
	resolved, err := request.resolve(path.Dir(rc.SourceFile()), rc)
	if err != nil {
		return "", err
	}
	return resolved.render(rc)
}

func parseIncludeRequest(rc render.Context) (includeRequest, error) {
	argsline, err := rc.ExpandTagArg()
	if err != nil {
		return includeRequest{}, err
	}
	args, err := ParseArgs(argsline)
	if err != nil {
		return includeRequest{}, err
	}
	if len(args.Args) != 1 {
		return includeRequest{}, fmt.Errorf("parse error")
	}
	include, err := args.EvalOptions(rc)
	if err != nil {
		return includeRequest{}, err
	}
	return includeRequest{filename: args.Args[0], include: include}, nil
}

func (tc tagContext) renderInclude(request includeRequest, rc render.Context) (s string, err error) {
	for _, dir := range tc.includeDirs {
		var resolved resolvedInclude
		resolved, err = request.resolve(dir, rc)
		if err == nil {
			s, err = resolved.render(rc)
		}
		if err == nil {
			return
		}
	}
	return
}

func (r includeRequest) resolve(dir string, rc render.Context) (resolvedInclude, error) {
	filename := filepath.Join(dir, r.filename)
	includeStack := getIncludeStack(rc)
	for _, entry := range includeStack {
		if entry.filename == filename && reflect.DeepEqual(entry.include, r.include) {
			return resolvedInclude{}, fmt.Errorf("include loop detected: %s", filename)
		}
	}
	stack := append(append([]includeStackEntry(nil), includeStack...), includeStackEntry{
		filename: filename,
		include:  r.include,
	})
	return resolvedInclude{filename: filename, include: r.include, stack: stack}, nil
}

func (r resolvedInclude) render(rc render.Context) (string, error) {
	vars := map[string]interface{}{
		"include":           r.include,
		"__include_stack__": r.stack,
	}
	return rc.RenderFile(r.filename, vars)
}

func getIncludeStack(rc render.Context) []includeStackEntry {
	if stack := rc.Get("__include_stack__"); stack != nil {
		if s, ok := stack.([]includeStackEntry); ok {
			return s
		}
	}
	return []includeStackEntry{}
}
