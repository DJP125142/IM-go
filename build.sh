#!/bin/bash

# 获取当前脚本所在的目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# 判断操作系统类型
OS=$(uname -s)
if [[ "${OS}" == "Darwin" ]]; then
    # Mac OS X
    # 编译项目
    go build -o "${SCRIPT_DIR}/IMserver"
elif [[ "${OS}" == "Linux" ]]; then
    # Linux
    # 编译项目
    go build -o "${SCRIPT_DIR}/IMserver"
elif [[ "${OS}" == "Windows" ]]; then
    # Windows
    # 编译项目
    go build -o "${SCRIPT_DIR}/IMserver.exe"
else
    echo "Unsupported operating system: ${OS}"
    exit 1
fi

# 切换到项目根目录
cd "${SCRIPT_DIR}"

# 启动服务
pkill IMserver
nohup ./IMserver >/dev/null 2>&1 &

# 输出成功信息
echo "IMserver服务已启动！"

# 结束脚本
exit 0
