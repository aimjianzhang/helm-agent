package command

import "helm-example/pkg/model"

type Func func(context *model.Context, cmd *model.Packet) ([]*model.Packet, *model.Packet)

type FuncMap map[string]Func

var Funcs = FuncMap{}

func (fs *FuncMap) Add(key string, f Func) {
	p := *fs
	p[key] = f
}

