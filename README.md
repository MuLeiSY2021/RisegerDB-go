# RisegerDB Go Edition

**Rapid Search of Geographic Database** — 基于 R-tree 空间索引的地理数据库系统。

从 [Java 版本](https://github.com/MuLeiSY2021/RisegerDB) 迁移而来，使用 Go 语言重写。

## 特性

- **R-tree / R\*-tree / STR R-tree** 空间索引，支持高效的地理空间查询
- **SQL-like 查询语言**，支持 SEARCH / WHERE / UPDATE / DELETE / CREATE
- **空间运算**：IN（包含）、OUT（排除）、RECT（矩形范围）、坐标点查询
- **HTTP REST API**，`curl` 即可交互，天然支持各种语言客户端
- **交互式 CLI 客户端**，彩色表格输出
- **WAL 预写日志**，保证崩溃恢复数据不丢
- **Protobuf 序列化**，紧凑的磁盘存储格式
- **geodata 工具链**，`.geodata` 二进制格式批量导入地理数据

## 快速开始

### 编译

```bash
go build -o riseger-server ./cmd/riseger-server
go build -o riseger-cli ./cmd/riseger-cli
go build -o riseger-geodata-gen ./cmd/riseger-geodata-gen
```

### 启动服务

```bash
./riseger-server --data ./data --port 12000
```

### 生成测试数据并导入

```bash
# 生成 .geodata 文件
./riseger-geodata-gen -o sample.geodata -db test_db -buildings 5

# 通过 CLI 导入
./riseger-cli --port 12000
RisegerDB> PRELOAD '/path/to/sample.geodata';
```

### CLI 查询

```bash
./riseger-cli --host localhost --port 12000
```

```sql
-- 创建数据库和模型
CREATE DATABASE 'my_db';
USE DATABASE my_db CREATE MAP 'world' NODESIZE 8 THRESHOLD 0.5;
USE DATABASE my_db CREATE MODEL 'building' PARENT 'field' PARAM name STRING PARAM area DOUBLE;

-- 查询
USE DATABASE my_db | MAP world SEARCH name, area;
USE DATABASE my_db | MAP world SEARCH name WHERE area > 100;
USE DATABASE my_db | MAP world SEARCH name WHERE area > 80 AND area < 200;
USE DATABASE my_db | MAP world SEARCH name WHERE IN RECT([100, 200], 50);

-- 更新
USE DATABASE my_db | MAP world UPDATE name = 'new_name' WHERE area > 500;

-- 删除
USE DATABASE my_db | MAP world DELETE FROM building WHERE area < 10;

-- 导入数据
PRELOAD '/path/to/data.geodata';

-- 元数据查询
GET DATABASES;
USE DATABASE my_db GET MAPS;
USE DATABASE my_db GET MODELS;
```

### HTTP API

```bash
# 健康检查
curl http://localhost:12000/health

# 执行查询
curl -X POST http://localhost:12000/query \
  -H "Content-Type: application/json" \
  -d '{"sql": "USE DATABASE test_db | MAP china SEARCH name, area WHERE area > 100"}'
```

响应格式：
```json
{
  "success": true,
  "columns": ["name", "area"],
  "rows": [
    {"name": "Building A", "area": 150.5},
    {"name": "Building B", "area": 200.0}
  ],
  "rowCount": 2,
  "time": "1.234ms"
}
```

## 项目结构

```
├── cmd/
│   ├── riseger-server/        # 服务端入口
│   ├── riseger-cli/           # 命令行客户端 (HTTP)
│   └── riseger-geodata-gen/   # 地理数据生成工具
├── internal/
│   ├── engine/                # 数据库引擎核心
│   ├── server/                # HTTP REST 服务器
│   ├── compile/               # 查询编译系统
│   │   ├── lexer/             # 词法分析器
│   │   ├── parser/            # 递归下降解析器
│   │   └── function/          # 执行引擎
│   ├── cache/                 # 内存缓存系统
│   ├── storage/               # 磁盘存储系统
│   ├── wal/                   # 预写日志系统
│   └── config/                # 配置管理
├── pkg/
│   ├── rtree/                 # R-tree 空间索引库
│   ├── protocol/              # 通信协议
│   └── geodata/               # .geodata 格式 (protobuf)
└── go.mod
```

## SQL 语法参考

详见 [`internal/compile/parser/parser.go`](internal/compile/parser/parser.go) 文件顶部的完整语法定义。

## License

MIT
