// Copyright(C) 2019 github.com/hidu  All Rights Reserved.
// Author: hidu (duv123+git@baidu.com)
// Date: 2019/12/21

package localmodule

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type config struct {
	// 域名对应目录层级
	DomainLevel map[string]int
}

var loadConfig = sync.OnceValue(readConfig)

// 加载用户自定义配置,配置内容为json，
// 比如：
// {"abc.com":3}
func readConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	confPath := filepath.Join(home, ".go_fmt", "local_module.json")
	_, err = os.Stat(confPath)
	// 若配置文件不存在则 跳过
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	confBuf, err := os.ReadFile(confPath)
	if err != nil {
		return err
	}
	var conf *config
	// 若配置解析失败则忽略掉
	if err = json.Unmarshal(confBuf, &conf); err != nil {
		return nil
	}

	return parserConfig4GoPath(conf)
}
