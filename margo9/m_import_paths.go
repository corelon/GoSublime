package main

import (
	"go/ast"
	"go/parser"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type mImportPaths struct {
	Fn  string
	Src string
	Env map[string]string
}

type mImportPathsDecl struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (m *mImportPaths) Call() (interface{}, string) {
	paths, _ := importPaths(m.Env)
	imports := []mImportPathsDecl{}
	_, af, err := parseAstFile(m.Fn, m.Src, parser.ImportsOnly)
	if err != nil {
		return M{}, err.Error()
	}

	if m.Fn != "" || m.Src != "" {
		for _, decl := range af.Decls {
			if gdecl, ok := decl.(*ast.GenDecl); ok && len(gdecl.Specs) > 0 {
				for _, spec := range gdecl.Specs {
					if ispec, ok := spec.(*ast.ImportSpec); ok {
						sd := mImportPathsDecl{
							Path: unquote(ispec.Path.Value),
						}
						if ispec.Name != nil {
							sd.Name = ispec.Name.String()
						}
						imports = append(imports, sd)
					}
				}
			}
		}
	}

	res := M{
		"imports": imports,
		"paths":   paths,
	}
	return res, ""
}

func init() {
	registry.Register("import_paths", func(_ *Broker) Caller {
		return &mImportPaths{
			Env: map[string]string{},
		}
	})
}

func importPaths(environ map[string]string) ([]string, error) {
	imports := []string{
		"unsafe",
	}
	paths := map[string]bool{}

	env := []string{
		environ["GOPATH"],
		environ["GOROOT"],
		os.Getenv("GOPATH"),
		os.Getenv("GOROOT"),
		runtime.GOROOT(),
	}
	for _, ent := range env {
		for _, path := range filepath.SplitList(ent) {
			if path != "" {
				paths[path] = true
			}
		}
	}

	seen := map[string]bool{}
	pfx := strings.HasPrefix
	sfx := strings.HasSuffix
	osArch := runtime.GOOS + "_" + runtime.GOARCH
	for root, _ := range paths {
		root = filepath.Join(root, "pkg", osArch)
		walkF := func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				p, e := filepath.Rel(root, p)
				if e == nil && sfx(p, ".a") {
					p := p[:len(p)-2]
					if !pfx(p, ".") && !pfx(p, "_") && !sfx(p, "_test") {
						p = path.Clean(filepath.ToSlash(p))
						if !seen[p] {
							seen[p] = true
							imports = append(imports, p)
						}
					}
				}
			}
			return nil
		}
		filepath.Walk(root, walkF)
	}
	return imports, nil
}
