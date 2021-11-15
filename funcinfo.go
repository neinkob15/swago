package swago

import (
	"go/parser"
	"go/token"
	"reflect"
	"runtime"
)

type FuncInfo struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary"`
	Security    []map[string][]string `json:"security"`
	Description string                `json:"description"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   Responses             `json:"responses"`
}
type Parameter struct {
	Name     string      `json:"name"`
	In       string      `json:"in"`
	Required bool        `json:"required"`
	Schema   ParamSchema `json:"schema"`
}
type ParamSchema struct {
	Type string   `json:"type"`
	Enum []string `json:"enum,omitempty"`
}
type RequestBody struct {
	Required    bool    `json:"required,omitempty"`
	Content     Content `json:"content"`
	Description string  `json:"description,omitempty"`
}
type Responses map[string]Response
type Response struct {
	Description string  `json:"description"`
	Content     Content `json:"content"`
}
type Content struct {
	ContentType ContentType `json:"application/json"`
}
type ContentType struct {
	Schema Schema `json:"schema"`
}
type Schema struct {
	Ref string `json:"$ref"`
}

func getCallerFrame(i interface{}) *runtime.Frame {
	pc := reflect.ValueOf(i).Pointer()
	frames := runtime.CallersFrames([]uintptr{pc})
	if frames == nil {
		return nil
	}
	frame, _ := frames.Next()
	if frame.Entry == 0 {
		return nil
	}
	return &frame
}

func getPkgName(file string) string {
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, file, nil, parser.PackageClauseOnly)
	if err != nil {
		return ""
	}
	if astFile.Name == nil {
		return ""
	}
	return astFile.Name.Name
}

func getFuncComment(file string, line int) string {
	fset := token.NewFileSet()

	astFile, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return ""
	}

	if len(astFile.Comments) == 0 {
		return ""
	}

	for _, cmt := range astFile.Comments {
		if fset.Position(cmt.End()).Line+1 == line {
			return cmt.Text()
		}
	}

	return ""
}
