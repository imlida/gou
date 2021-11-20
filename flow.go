package gou

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/kun/exception"
)

// Flows 已加载工作流列表
var Flows = map[string]*Flow{}

// LoadFlow 载入数据接口
func LoadFlow(source string, name string) *Flow {
	var input io.Reader = nil
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		file, err := os.Open(filename)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		defer file.Close()
		input = file
	} else {
		input = strings.NewReader(source)
	}

	flow := Flow{
		Name:    name,
		Source:  source,
		Scripts: map[string]string{},
	}
	err := helper.UnmarshalFile(input, &flow)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	flow.Prepare()
	Flows[name] = &flow
	return Flows[name]
}

// Prepare 预加载 Query DSL
func (flow *Flow) Prepare() {

	if flow.Scripts == nil {
		flow.Scripts = map[string]string{}
	}

	for i, node := range flow.Nodes {
		if node.Query == nil {
			continue
		}

		if node.Engine == "" {
			log.Printf("Node %s: 未指定数据查询分析引擎", node.Name)
			continue
		}

		if engine, has := Engines[node.Engine]; has {
			var err error
			flow.Nodes[i].DSL, err = engine.Load(node.Query)
			if err != nil {
				log.Printf("Node %s: %s 数据分析查询解析错误", node.Name, node.Engine)
				log.Println(node.Query)
			}
			continue
		}
		log.Printf("Node %s: %s 数据分析引擎尚未注册", node.Name, node.Engine)
	}
}

// LoadScript 载入脚本
func (flow *Flow) LoadScript(source string, name string) *Flow {
	var input io.Reader = nil
	name = fmt.Sprintf("flows.%s.%s", flow.Name, name)
	if strings.HasPrefix(source, "file://") {
		filename := strings.TrimPrefix(source, "file://")
		err := JavaScriptVM.Load(filename, name)
		if err != nil {
			log.Printf("加载数据脚本失败 %s: %s", filename, name)
		}
	} else {
		input = strings.NewReader(source)
		err := JavaScriptVM.LoadSource("", input, name)
		if err != nil {
			log.Printf("加载数据脚本失败 %s", name)
		}
	}
	flow.Scripts[name] = source
	return flow
}

// Reload 重新载入API
func (flow *Flow) Reload() *Flow {
	new := LoadFlow(flow.Source, flow.Name)
	for name, source := range flow.Scripts {
		new.LoadScript(source, name)
	}
	flow = new
	Flows[flow.Name] = new
	return flow
}

// WithSID 设定会话ID
func (flow *Flow) WithSID(sid string) *Flow {
	flow.Sid = sid
	return flow
}

// WithGlobal 设定全局变量
func (flow *Flow) WithGlobal(global map[string]interface{}) *Flow {
	flow.Global = global
	return flow
}

// SelectFlow 读取已加载Flow
func SelectFlow(name string) *Flow {
	flow, has := Flows[name]
	if !has {
		exception.New(
			fmt.Sprintf("Flow:%s; 尚未加载", name),
			400,
		).Throw()
	}
	return flow
}
