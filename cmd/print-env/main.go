package main

import (
	"flag"
	"fmt"
	"go/types"
	"os"
	"reflect"
	"slices"
	"strings"
	"golang.org/x/tools/go/packages"
	"cmp"
)

type fieldEntry struct {
	Key   string
	Group string
	Tag   string
}

func main() {
	flag.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "usage: print-env <pkg>.<Type>   e.g. print-env ./internal/config.Config")
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	arg := flag.Arg(0)
	i := strings.LastIndex(arg, ".")
	if i < 0 || i == len(arg)-1 {
		die("argument must be <pkg>.<Type>, got %q", arg)
	}
	pkgPattern, typeName := arg[:i], arg[i+1:]
	if pkgPattern == "" {
		pkgPattern = "."
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedSyntax |
			packages.NeedImports |
			packages.NeedDeps,
	}

	pkgs, err := packages.Load(cfg, pkgPattern)
	if err != nil {
		die("load error: %v", err)
	}

	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	if len(pkgs) == 0 {
		die("no package found")
	}

	pkg := pkgs[0]

	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		die("type %q not found in package %s", typeName, pkg.PkgPath)
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		die("%q is not a named type", typeName)
	}

	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		die("%q is not a struct", typeName)
	}

	visiting := map[types.Type]struct{}{named: {}}
	fields := flattenStruct(st, visiting, "", "", pkg.Types)

	printEnv(fields)
}

func flattenStruct(
	st *types.Struct,
	visiting map[types.Type]struct{},
	prefix string,
	group string,
	rootPkg *types.Package,
) []fieldEntry {
	var out []fieldEntry

	for i := range st.NumFields() {
		field := st.Field(i)
		tag := st.Tag(i)
		fieldType := deref(field.Type())

		if nested, _ := fieldType.Underlying().(*types.Struct); nested != nil {
			if _, ok := visiting[fieldType]; ok {
				continue
			}
			name := field.Name()
			if field.Embedded() {
				name = embeddedCommentName(field, tag, rootPkg)
			}

			childPrefix := prefix
			if p := reflect.StructTag(tag).Get("envPrefix"); p != "" && p != "-" {
				childPrefix = joinNonEmpty(prefix, p, "_")
			}
			childGroup := joinNonEmpty(group, name, ".")

			visiting[fieldType] = struct{}{}
			out = append(out, flattenStruct(nested, visiting, childPrefix, childGroup, rootPkg)...)
			delete(visiting, fieldType)
			continue
		}

		fieldEnv := reflect.StructTag(tag).Get("env")
		if fieldEnv == "" || fieldEnv == "-" {
			continue
		}

		out = append(out, fieldEntry{
			Key:   joinNonEmpty(prefix, fieldEnv, "_"),
			Group: group,
			Tag:   tag,
		})
	}

	return out
}

func joinNonEmpty(a, b, sep string) string {
	if a == "" {
		return b
	}
	return a + sep + b
}

func die(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func deref(t types.Type) types.Type {
	if ptr, ok := t.(*types.Pointer); ok {
		return ptr.Elem()
	}
	return t
}

func printEnv(fields []fieldEntry) {
	slices.SortStableFunc(fields, func(i, j fieldEntry) int {
		return cmp.Or(strings.Compare(i.Group, j.Group), strings.Compare(i.Key, j.Key))
	})

	seenKeys := make(map[string]struct{})
	lastGroup := ""

	for _, f := range fields {
		if _, ok := seenKeys[f.Key]; ok {
			continue
		}
		seenKeys[f.Key] = struct{}{}

		if f.Group != "" && f.Group != lastGroup {
			fmt.Println()
			fmt.Printf("# %s\n", f.Group)
			lastGroup = f.Group
		}

		fmt.Printf("%s=%s\n", f.Key, reflect.StructTag(f.Tag).Get("envDefault"))
	}
}

func embeddedCommentName(field *types.Var, tag string, rootPkg *types.Package) string {
	t := deref(field.Type())

	named, ok := t.(*types.Named)
	if !ok {
		return field.Name()
	}

	obj := named.Obj()
	if obj == nil {
		return field.Name()
	}

	typeName := obj.Name()
	typePkg := obj.Pkg()

	if typePkg != nil && rootPkg != nil && typePkg.Path() != rootPkg.Path() {
		return typePkg.Name() + "." + typeName
	}

	if prefix := reflect.StructTag(tag).Get("envPrefix"); prefix != "" && prefix != "-" {
		return titleEnvPrefix(prefix)
	}

	return typeName
}

func titleEnvPrefix(prefix string) string {
	parts := strings.Split(prefix, "_")

	for i, part := range parts {
		part = strings.ToLower(part)
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}

	return strings.Join(parts, "")
}
