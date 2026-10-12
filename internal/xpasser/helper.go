// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/8/18

package xpasser

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/fsgo/go_fmt/internal/common"
)

func TypeOf(req *common.Request, e ast.Expr) (types.Type, error) {
	if req.TypesInfo == nil {
		return nil, fmt.Errorf("type information unavailable for %s", req.FileName)
	}
	vt := req.TypesInfo.TypeOf(e)
	if vt != nil {
		return vt, nil
	}
	return nil, fmt.Errorf("type not found for %v", e)
}
