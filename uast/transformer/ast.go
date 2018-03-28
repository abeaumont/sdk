package transformer

import (
	"gopkg.in/bblfsh/sdk.v1/uast"
	"gopkg.in/bblfsh/sdk.v1/uast/role"
)

// SavePosOffset makes an operation that describes a uast.Position object with Offset field set to a named variable.
func SavePosOffset(vr string) Op {
	return TypedObj(uast.TypePosition, map[string]Op{
		uast.KeyPosOff: Var(vr),
	})
}

// SavePosLine makes an operation that describes a uast.Position object with Line field set to a named variable.
func SavePosLine(vr string) Op {
	return TypedObj(uast.TypePosition, map[string]Op{
		uast.KeyPosLine: Var(vr),
	})
}

// SavePosCol makes an operation that describes a uast.Position object with Col field set to a named variable.
func SavePosCol(vr string) Op {
	return TypedObj(uast.TypePosition, map[string]Op{
		uast.KeyPosCol: Var(vr),
	})
}

// Roles makes an operation that will check/construct a list of roles.
func Roles(roles ...role.Role) ArrayOp {
	arr := make([]Op, 0, len(roles))
	for _, r := range roles {
		arr = append(arr, Is(uast.String(r.String())))
	}
	return Arr(arr...)
}

// AppendRoles can be used to append more roles to an output of a specific operation.
func AppendRoles(old ArrayOp, roles ...role.Role) ArrayOp {
	if len(roles) == 0 {
		return old
	}
	return AppendArr(old, Roles(roles...))
}

// ASTMap is a helper for creating a two-way mapping between AST and its normalized form.
func ASTMap(name string, native, norm Op) Mapping {
	return Mapping{
		Name: name,
		Steps: []Step{
			{Name: "native", Op: native},
			{Name: "norm", Op: norm},
		},
	}
}

// RolesField will create a roles field that appends provided roles to existing ones.
// In case no roles are provided, it will save existing roles, if any.
func RolesField(vr string, roles ...role.Role) Field {
	return RolesFieldOp(vr, nil, roles...)
}

// RolesFieldOp is like RolesField but allows to specify custom roles op to use.
func RolesFieldOp(vr string, op ArrayOp, roles ...role.Role) Field {
	if len(roles) == 0 && op == nil {
		return Field{
			Name:     uast.KeyRoles,
			Op:       Var(vr),
			Optional: vr + "_exists",
		}
	}
	var rop ArrayOp
	if len(roles) != 0 && op != nil {
		rop = AppendRoles(op, roles...)
	} else if op != nil {
		rop = op
	} else {
		rop = Roles(roles...)
	}
	return Field{
		Name: uast.KeyRoles,
		Op: If(vr+"_exists",
			Append(NotEmpty(Var(vr)), rop),
			rop,
		),
	}
}

// ASTObjectLeft construct a native AST shape for a given type name.
func ASTObjectLeft(typ string, ast ObjectOp) ObjectOp {
	a := ast.Object()
	if _, ok := a.GetField(uast.KeyRoles); ok {
		panic("unexpected roles filed")
	}
	a.SetField(uast.KeyType, String(typ))
	a.SetFieldObj(RolesField(typ + "_roles"))
	return Part("_", a)
}

// ASTObjectRight constructs an annotated native AST shape with specific roles.
func ASTObjectRight(typ string, norm ObjectOp, rop ArrayOp, roles ...role.Role) ObjectOp {
	b := norm.Object()
	if _, ok := b.GetField(uast.KeyRoles); ok {
		panic("unexpected roles filed")
	}
	b.SetField(uast.KeyType, String(typ)) // TODO: "<lang>:" namespace
	// it merges 3 slices:
	// 1) roles saved from left side (if any)
	// 2) static roles from arguments
	// 3) roles from conditional operation
	b.SetFieldObj(RolesFieldOp(typ+"_roles", rop, roles...))
	return Part("_", b)
}

// ObjectRoles creates a shape that adds additional roles to an object.
// Should only be used in other object fields, since it does not set any type constraints.
func ObjectRoles(vr string, roles ...role.Role) Op {
	return Part(vr, Fields{
		RolesField(vr+"_roles", roles...),
	})
}

// OptObjectRoles is like ObjectRoles, but marks an object as optional.
func OptObjectRoles(vr string, roles ...role.Role) Op {
	return Opt(vr+"_set", ObjectRoles(vr, roles...))
}
