package gotsrpc

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"os"
	"reflect"
	"strings"
)

var ReaderTrace = false

func readStructs(pkg *ast.Package, packageName string) (structs map[string]*Struct, err error) {
	structs = map[string]*Struct{}
	trace("reading files in package", packageName)
	for _, file := range pkg.Files {
		//readFile(filename, file)

		err = extractStructs(file, packageName, structs)
		if err != nil {
			return
		}
	}
	return
}

func trace(args ...interface{}) {
	if ReaderTrace {
		fmt.Fprintln(os.Stderr, args...)
	}
}
func jsonTrace(args ...interface{}) {
	if ReaderTrace {
		for _, arg := range args {
			jsonBytes, jsonErr := json.MarshalIndent(arg, "", "	")
			if jsonErr != nil {
				trace(arg)
				continue
			}
			trace(string(jsonBytes))
		}
	}
}

func extractJSONInfo(tag string) *JSONInfo {
	t := reflect.StructTag(tag)
	jsonTagString := t.Get("json")
	if len(jsonTagString) == 0 {
		return nil
	}
	jsonTagParts := strings.Split(jsonTagString, ",")

	name := ""
	ignore := false
	omit := false
	forceStringType := false
	cleanParts := []string{}
	for _, jsonTagPart := range jsonTagParts {
		cleanParts = append(cleanParts, strings.TrimSpace(jsonTagPart))
	}
	switch len(cleanParts) {
	case 1:
		switch cleanParts[0] {
		case "-":
			ignore = true
		default:
			name = cleanParts[0]
		}
	case 2:
		if len(cleanParts[0]) > 0 {
			name = cleanParts[0]
		}
		switch cleanParts[1] {
		case "omitempty":
			omit = true
		case "string":
			forceStringType = true
		}
	}
	return &JSONInfo{
		Name:            name,
		OmitEmpty:       omit,
		ForceStringType: forceStringType,
		Ignore:          ignore,
	}
}

func getScalarFromAstIdent(ident *ast.Ident) ScalarType {
	switch ident.Name {
	case "string":
		return ScalarTypeString
	case "bool":
		return ScalarTypeBool
	case "int", "int32", "int64", "float", "float32", "float64":
		return ScalarTypeNumber
	default:
		return ScalarTypeNone
	}
}

func getTypesFromAstType(ident *ast.Ident) (structType string, scalarType ScalarType) {
	scalarType = getScalarFromAstIdent(ident)
	switch scalarType {
	case ScalarTypeNone:
		structType = ident.Name
	}
	return
}

func readAstType(v *Value, fieldIdent *ast.Ident) {
	_, scalarType := getTypesFromAstType(fieldIdent)
	v.ScalarType = scalarType

	// if len(structType) > 0 {
	// 	v.StructType = &StructType{
	// 		Name: structType,
	// 		//Package: fieldIdent.String(),
	// 	}
	// 	trace("----------------->", fieldIdent)
	// } else {
	v.GoScalarType = fieldIdent.Name
	// }
}

func readAstStarExpr(v *Value, starExpr *ast.StarExpr, fileImports fileImportSpecMap) {
	v.IsPtr = true
	switch reflect.ValueOf(starExpr.X).Type().String() {
	case "*ast.Ident":
		ident := starExpr.X.(*ast.Ident)
		v.StructType = &StructType{
			Name:    ident.Name,
			Package: fileImports.getPackagePath(""),
		}
	case "*ast.StructType":
		// nested anonymous
		readAstStructType(v, starExpr.X.(*ast.StructType), fileImports)
	case "*ast.SelectorExpr":
		readAstSelectorExpr(v, starExpr.X.(*ast.SelectorExpr), fileImports)
	default:
		trace("a pointer on what", reflect.ValueOf(starExpr.X).Type().String())
	}
}

func readAstArrayType(v *Value, arrayType *ast.ArrayType, fileImports fileImportSpecMap) {
	switch reflect.ValueOf(arrayType.Elt).Type().String() {
	case "*ast.StarExpr":
		readAstStarExpr(v, arrayType.Elt.(*ast.StarExpr), fileImports)
	default:
		trace("array type elt", reflect.ValueOf(arrayType.Elt).Type().String())
	}
}

func readAstMapType(m *Map, mapType *ast.MapType, fileImports fileImportSpecMap) {
	trace("		map key", mapType.Key, reflect.ValueOf(mapType.Key).Type().String())
	trace("		map value", mapType.Value, reflect.ValueOf(mapType.Value).Type().String())
	// key
	switch reflect.ValueOf(mapType.Key).Type().String() {
	case "*ast.Ident":
		_, scalarType := getTypesFromAstType(mapType.Key.(*ast.Ident))
		m.KeyType = string(scalarType)
	}
	// value
	m.Value.loadExpr(mapType.Value, fileImports)
}

func readAstSelectorExpr(v *Value, selectorExpr *ast.SelectorExpr, fileImports fileImportSpecMap) {
	switch reflect.ValueOf(selectorExpr.X).Type().String() {
	case "*ast.Ident":
		// that could be the package name
		selectorIdent := selectorExpr.X.(*ast.Ident)
		packageName := selectorIdent.Name
		v.StructType = &StructType{
			Package: fileImports.getPackagePath(packageName),
			Name:    selectorExpr.Sel.Name,
		}
	default:
		trace("selectorExpr.Sel !?", selectorExpr.X, reflect.ValueOf(selectorExpr.X).Type().String())
	}
}

func readAstStructType(v *Value, structType *ast.StructType, fileImports fileImportSpecMap) {
	v.Struct = &Struct{}
	v.Struct.Fields = readFieldList(structType.Fields.List, fileImports)
}

func (v *Value) loadExpr(expr ast.Expr, fileImports fileImportSpecMap) {
	//fmt.Println(field.Names[0].Name, field.Type, reflect.ValueOf(field.Type).Type().String())
	switch reflect.ValueOf(expr).Type().String() {
	case "*ast.ArrayType":
		fieldArray := expr.(*ast.ArrayType)
		v.Array = &Array{Value: &Value{}}
		switch reflect.ValueOf(fieldArray.Elt).Type().String() {
		case "*ast.Ident":
			readAstType(v.Array.Value, fieldArray.Elt.(*ast.Ident))
		case "*ast.StarExpr":
			readAstStarExpr(v.Array.Value, fieldArray.Elt.(*ast.StarExpr), fileImports)
		case "*ast.ArrayType":
			readAstArrayType(v.Array.Value, fieldArray.Elt.(*ast.ArrayType), fileImports)
		case "*ast.MapType":
			v.Array.Value.Map = &Map{
				Value: &Value{},
			}
			readAstMapType(v.Array.Value.Map, fieldArray.Elt.(*ast.MapType), fileImports)
		default:
			trace("---------------------> array of", reflect.ValueOf(fieldArray.Elt).Type().String())
		}
	case "*ast.Ident":
		fieldIdent := expr.(*ast.Ident)
		readAstType(v, fieldIdent)
	case "*ast.StarExpr":
		// a pointer on sth
		readAstStarExpr(v, expr.(*ast.StarExpr), fileImports)
	case "*ast.MapType":
		v.Map = &Map{
			Value: &Value{},
		}
		readAstMapType(v.Map, expr.(*ast.MapType), fileImports)
	case "*ast.SelectorExpr":
		readAstSelectorExpr(v, expr.(*ast.SelectorExpr), fileImports)
	case "*ast.StructType":
		readAstStructType(v, expr.(*ast.StructType), fileImports)
	default:
		trace("what kind of field ident would that be ?!", reflect.ValueOf(expr).Type().String())
	}
}

func readField(astField *ast.Field, fileImports fileImportSpecMap) (name string, v *Value, jsonInfo *JSONInfo) {
	name = ""
	if len(astField.Names) > 0 {
		name = astField.Names[0].Name
	}

	trace("	reading field with name", name, "of type", astField.Type)
	v = &Value{}
	v.loadExpr(astField.Type, fileImports)
	if astField.Tag != nil {
		jsonInfo = extractJSONInfo(astField.Tag.Value[1 : len(astField.Tag.Value)-1])
	}
	return
}

func readFieldList(fieldList []*ast.Field, fileImports fileImportSpecMap) (fields []*Field) {
	fields = []*Field{}
	for _, field := range fieldList {
		name, value, jsonInfo := readField(field, fileImports)
		if len(name) == 0 {
			trace("i do not understand this one", field, name, value, jsonInfo)
			continue
		}
		if strings.Compare(strings.ToLower(name[:1]), name[:1]) == 0 {
			continue
		}

		if value != nil {
			fields = append(fields, &Field{
				Name:     name,
				Value:    value,
				JSONInfo: jsonInfo,
			})
		}
	}
	return
}

func extractStructs(file *ast.File, packageName string, structs map[string]*Struct) error {

	trace("reading file", file.Name.Name)
	//for _, imp := range file.Imports {
	//	fmt.Println("import", imp.Name, imp.Path)
	//}
	fileImports := getFileImports(file, packageName)
	for name, obj := range file.Scope.Objects {
		//fmt.Println(name, obj.Kind, obj.Data)
		if obj.Kind == ast.Typ && obj.Decl != nil {
			//ast.StructType
			structName := packageName + "." + name
			structs[structName] = &Struct{
				Name:    name,
				Fields:  []*Field{},
				Package: packageName,
			}
			if reflect.ValueOf(obj.Decl).Type().String() == "*ast.TypeSpec" {
				typeSpec := obj.Decl.(*ast.TypeSpec)
				typeSpecRefl := reflect.ValueOf(typeSpec.Type)
				if typeSpecRefl.Type().String() == "*ast.StructType" {
					structType := typeSpec.Type.(*ast.StructType)
					trace("StructType", obj.Name)
					structs[structName].Fields = readFieldList(structType.Fields.List, fileImports)
				} else {
					//	fmt.Println("	what would that be", typeSpecRefl.Type().String())
				}
			} else {
				//fmt.Println("	!!!!!!!!!!!!!!!!!!!!!!!!!!!! not a type spec", r.Type().String())
			}
		} else if obj.Kind == ast.Fun {
			//fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> a fucking func / method", obj.Kind, obj)
		} else {
			//fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>", obj.Kind, obj)
		}
	}
	return nil
}
