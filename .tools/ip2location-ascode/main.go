package main

import (
	"bytes"
	"compress/flate"
	"encoding/ascii85"
	"encoding/hex"
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

var (
	pkgname string
	output  string
)

func main() {
	if len(os.Args) < 2 {
		stop("ip2location IP2LOCATION-LITE-DB1.IPV6.BIN IP2LOCATION-LITE-DB1.BIN")
	}

	flag.StringVar(&pkgname, "pkgname", "lookup", "package name")
	flag.StringVar(&output, "output", "lookup/assets.go", "output file")
	flag.Parse()

	//

	var constants []ast.Decl
	var clauses []ast.Stmt

	//

	for _, filename := range os.Args[1:] {
		constant, clause, err := craft(filename)
		if err != nil {
			stop(filename, err)
		}

		constants = append(constants, constant)
		clauses = append(clauses, clause)
	}

	//

	f, err := os.Create(output)
	if err != nil {
		stop(err)
	}

	fset := token.NewFileSet()
	err = printer.Fprint(f, fset, astfile(constants, clauses))
	if err != nil {
		stop("render ast", err)
	}

	err = f.Sync()
	if err != nil {
		stop(err)
	}
}

func stop(args ...interface{}) {
	fmt.Println(args...)
	os.Exit(1)
}

func compress(payload []byte) (string, int, error) {
	var best string
	var level int
	var buf bytes.Buffer

	for i := flate.BestSpeed; i <= flate.BestCompression; i++ {
		buf.Reset()

		codec := ascii85.NewEncoder(&buf)

		w, err := flate.NewWriter(codec, i)
		if err != nil {
			return "", 0, err
		}

		if _, err = w.Write(payload); err != nil {
			return "", 0, err
		}

		if err = w.Flush(); err != nil {
			return "", 0, err
		}

		if err = w.Close(); err != nil {
			return "", 0, err
		}

		if err = codec.Close(); err != nil {
			return "", 0, err
		}

		if buf.Len() < len(best) || i == flate.BestSpeed {
			level = i
			best = buf.String()
		}
	}

	return best, level, nil
}

func decompress(payload []byte) ([]byte, error) {
	r := ascii85.NewDecoder(bytes.NewReader(payload))
	rc := flate.NewReader(r)
	defer rc.Close()

	return io.ReadAll(rc)
}

func astfile(constants []ast.Decl, clauses []ast.Stmt) *ast.File {
	// Add default clause
	clauses = append(clauses, &ast.CaseClause{
		Body: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.Ident{
						Name: "nil",
					},
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.Ident{
								Name: "fmt",
							},
							Sel: &ast.Ident{
								Name: "Errorf",
							},
						},
						Args: []ast.Expr{
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: `"%s: %w"`,
							},
							&ast.Ident{
								Name: "name",
							},
							&ast.SelectorExpr{
								X: &ast.Ident{
									Name: "fs",
								},
								Sel: &ast.Ident{
									Name: "ErrNotExist",
								},
							},
						},
					},
				},
			},
		},
	})

	// Create base file with clauses added
	f := &ast.File{
		Name: &ast.Ident{
			Name: pkgname,
		},
		Decls: []ast.Decl{
			&ast.GenDecl{
				Tok: token.IMPORT,
				Specs: []ast.Spec{
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"bytes"`,
						},
					},
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"compress/flate"`,
						},
					},
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"encoding/ascii85"`,
						},
					},
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"fmt"`,
						},
					},
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"io"`,
						},
					},
					&ast.ImportSpec{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"io/fs"`,
						},
					},
				},
			},
			&ast.FuncDecl{
				Name: &ast.Ident{
					Name: "database",
				},
				Type: &ast.FuncType{
					Params: &ast.FieldList{
						List: []*ast.Field{
							&ast.Field{
								Names: []*ast.Ident{
									&ast.Ident{
										Name: "name",
									},
								},
								Type: &ast.Ident{
									Name: "string",
								},
							},
						},
					},
					Results: &ast.FieldList{
						List: []*ast.Field{
							&ast.Field{
								Type: &ast.ArrayType{
									Elt: &ast.Ident{
										Name: "byte",
									},
								},
							},
							&ast.Field{
								Type: &ast.Ident{
									Name: "error",
								},
							},
						},
					},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.SwitchStmt{
							Tag: &ast.Ident{
								Name: "name",
							},
							Body: &ast.BlockStmt{
								List: clauses,
							},
						},
					},
				},
			},
		},
	}

	// Add tooling
	f.Decls = append(f.Decls, decompressAST())
	f.Decls = append(f.Decls, rcatAST()...)

	// Add constants
	f.Decls = append(f.Decls, constants...)

	return f
}

func craft(filename string) (*ast.GenDecl, *ast.CaseClause, error) {
	payload, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	s, level, err := compress(payload)
	if err != nil {
		return nil, nil, err
	}

	filename = filepath.Base(filename)
	fmt.Printf("%s: %d Bytes at level %d\n", filename, len(s), level)

	//
	//

	name := "const" + hex.EncodeToString([]byte(filename))

	constant := &ast.GenDecl{
		Tok: token.CONST,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{
					&ast.Ident{
						Name: name,
					},
				},
				Values: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: strconv.Quote(s),
					},
				},
			},
		},
	}

	clause := &ast.CaseClause{
		List: []ast.Expr{
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: strconv.Quote(filename),
			},
		},
		Body: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.Ident{
							Name: "decompress",
						},
						Args: []ast.Expr{
							&ast.CallExpr{
								Fun: &ast.ArrayType{
									Elt: &ast.Ident{
										Name: "byte",
									},
								},
								Args: []ast.Expr{
									&ast.Ident{
										Name: name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return constant, clause, nil
}

func decompressAST() ast.Decl {
	return &ast.FuncDecl{
		Name: &ast.Ident{
			Name: "decompress",
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Names: []*ast.Ident{
							&ast.Ident{
								Name: "payload",
							},
						},
						Type: &ast.ArrayType{
							Elt: &ast.Ident{
								Name: "byte",
							},
						},
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Type: &ast.ArrayType{
							Elt: &ast.Ident{
								Name: "byte",
							},
						},
					},
					&ast.Field{
						Type: &ast.Ident{
							Name: "error",
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.Ident{
							Name: "r",
						},
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X: &ast.Ident{
									Name: "ascii85",
								},
								Sel: &ast.Ident{
									Name: "NewDecoder",
								},
							},
							Args: []ast.Expr{
								&ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X: &ast.Ident{
											Name: "bytes",
										},
										Sel: &ast.Ident{
											Name: "NewReader",
										},
									},
									Args: []ast.Expr{
										&ast.Ident{
											Name: "payload",
										},
									},
								},
							},
						},
					},
				},
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						&ast.Ident{
							Name: "rc",
						},
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X: &ast.Ident{
									Name: "flate",
								},
								Sel: &ast.Ident{
									Name: "NewReader",
								},
							},
							Args: []ast.Expr{
								&ast.Ident{
									Name: "r",
								},
							},
						},
					},
				},
				&ast.DeferStmt{
					Call: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X: &ast.Ident{
								Name: "rc",
							},
							Sel: &ast.Ident{
								Name: "Close",
							},
						},
					},
				},
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X: &ast.Ident{
									Name: "io",
								},
								Sel: &ast.Ident{
									Name: "ReadAll",
								},
							},
							Args: []ast.Expr{
								&ast.Ident{
									Name: "rc",
								},
							},
						},
					},
				},
			},
		},
	}
}

func rcatAST() []ast.Decl {
	return []ast.Decl{
		&ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: &ast.Ident{
						Name: "rcat",
					},
					Type: &ast.StructType{
						Fields: &ast.FieldList{
							List: []*ast.Field{
								&ast.Field{
									Type: &ast.SelectorExpr{
										X: &ast.Ident{
											Name: "bytes",
										},
										Sel: &ast.Ident{
											Name: "Reader",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		&ast.FuncDecl{
			Name: &ast.Ident{
				Name: "newrcat",
			},
			Type: &ast.FuncType{
				Params: &ast.FieldList{
					List: []*ast.Field{
						&ast.Field{
							Names: []*ast.Ident{
								&ast.Ident{
									Name: "b",
								},
							},
							Type: &ast.ArrayType{
								Elt: &ast.Ident{
									Name: "byte",
								},
							},
						},
					},
				},
				Results: &ast.FieldList{
					List: []*ast.Field{
						&ast.Field{
							Type: &ast.StarExpr{
								X: &ast.Ident{
									Name: "rcat",
								},
							},
						},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.UnaryExpr{
								Op: token.AND,
								X: &ast.CompositeLit{
									Type: &ast.Ident{
										Name: "rcat",
									},
									Elts: []ast.Expr{
										&ast.KeyValueExpr{
											Key: &ast.Ident{
												Name: "Reader",
											},
											Value: &ast.StarExpr{
												X: &ast.CallExpr{
													Fun: &ast.SelectorExpr{
														X: &ast.Ident{
															Name: "bytes",
														},
														Sel: &ast.Ident{
															Name: "NewReader",
														},
													},
													Args: []ast.Expr{
														&ast.Ident{
															Name: "b",
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		//
		// Close
		//
		&ast.FuncDecl{
			Recv: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Names: []*ast.Ident{
							&ast.Ident{
								Name: "r",
							},
						},
						Type: &ast.StarExpr{
							X: &ast.Ident{
								Name: "rcat",
							},
						},
					},
				},
			},
			Name: &ast.Ident{
				Name: "Close",
			},
			Type: &ast.FuncType{
				Results: &ast.FieldList{
					List: []*ast.Field{
						&ast.Field{
							Type: &ast.Ident{
								Name: "error",
							},
						},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.Ident{
								Name: "nil",
							},
						},
					},
				},
			},
		},
		//
		// ReadAt
		//
		&ast.FuncDecl{
			Recv: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Names: []*ast.Ident{
							&ast.Ident{
								Name: "r",
							},
						},
						Type: &ast.StarExpr{
							X: &ast.Ident{
								Name: "rcat",
							},
						},
					},
				},
			},
			Name: &ast.Ident{
				Name: "ReadAt",
			},
			Type: &ast.FuncType{
				Params: &ast.FieldList{
					List: []*ast.Field{
						&ast.Field{
							Names: []*ast.Ident{
								&ast.Ident{
									Name: "b",
								},
							},
							Type: &ast.ArrayType{
								Elt: &ast.Ident{
									Name: "byte",
								},
							},
						},
						&ast.Field{
							Names: []*ast.Ident{
								&ast.Ident{
									Name: "off",
								},
							},
							Type: &ast.Ident{
								Name: "int64",
							},
						},
					},
				},
				Results: &ast.FieldList{
					List: []*ast.Field{
						&ast.Field{
							Names: []*ast.Ident{
								&ast.Ident{
									Name: "n",
								},
							},
							Type: &ast.Ident{
								Name: "int",
							},
						},
						&ast.Field{
							Doc: nil,
							Names: []*ast.Ident{
								&ast.Ident{
									Name: "err",
								},
							},
							Type: &ast.Ident{
								Name: "error",
							},
						},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X: &ast.SelectorExpr{
										X: &ast.Ident{
											Name: "r",
										},
										Sel: &ast.Ident{
											Name: "Reader",
										},
									},
									Sel: &ast.Ident{
										Name: "ReadAt",
									},
								},
								Args: []ast.Expr{
									&ast.Ident{
										Name: "b",
									},
									&ast.Ident{
										Name: "off",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
