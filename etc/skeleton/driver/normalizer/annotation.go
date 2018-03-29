package normalizer

import (
	"gopkg.in/bblfsh/sdk.v1/uast"
	"gopkg.in/bblfsh/sdk.v1/uast/role"
	. "gopkg.in/bblfsh/sdk.v1/uast/transformer"
	"gopkg.in/bblfsh/sdk.v1/uast/transformer/positioner"
)

// Native is the of list `transformer.Transformer` to apply to a native AST.
// To learn more about the Transformers and the available ones take a look to:
// https://godoc.org/gopkg.in/bblfsh/sdk.v1/uast/transformer
var Native = Transformers([][]Transformer{
	{
		// ObjectToNode defines how to normalize common fields of native AST
		// (like node type, token, positional information).
		//
		// https://godoc.org/gopkg.in/bblfsh/sdk.v1/uast#ObjectToNode
		ObjectToNode{
			InternalTypeKey: "...", // native AST type key name
		},
	},
	// The main block of transformation rules.
	Annotations,
	{
		// RolesDedup is used to remove duplicate roles assigned by multiple
		// transformation rules.
		RolesDedup{},
	},
}...)

// Code is a special block of transformations that are applied at the end
// and can access original source code file. It can be used to improve or
// fix positional information.
//
// https://godoc.org/gopkg.in/bblfsh/sdk.v1/uast/transformer/positioner
var Code = []CodeTransformer{
	positioner.NewFillLineColFromOffset(),
}

// mapAST is a helper for describing a single AST transformation for a given node type.
func mapAST(typ string, ast, norm ObjectOp, roles ...role.Role) Mapping {
	return ASTMap(typ,
		ASTObjectLeft(typ, ast),
		ASTObjectRight(typ, norm, nil, roles...),
	)
}

// Annotations is a list of individual transformations to annotate a native AST with roles.
var Annotations = []Transformer{
	ASTMap("unannotated", Obj{}, Obj{
		uast.KeyRoles: Roles(role.Unannotated),
	}),
}
