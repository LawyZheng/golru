# GoLRU Cache [![Build Status](https://api.travis-ci.com/LawyZheng/golru.svg?branch=master)]()

基于 [Concurrent-Map](http://github.com/orcaman/concurrent-map)实现的一个轻量级高性能LRU缓存库。该缓存库有如下特点：
- [X] 近似LRU缓存算法。
- [X] 采用了泛型设计，告别对象序列化或者接口断言。

## 用法

导入包:

```go
import (
	"github.com/lawyzheng/golru"
)

```

```bash
go get "github.com/lawyzheng/golru"
```

## 示例

```go

	// 创建一个新的 cache.
	c := golru.New[string](0)

	// 设置变量m一个键为“foo”值为“bar”键值对
	c.Set("foo", "bar")

	// 从m中获取指定键值.
	bar, ok := c.Get("foo")

	// 删除键为“foo”的项
	c.Remove("foo")

```

更多使用示例请查看`lru_test.go`.

运行测试:

```bash
go test "github.com/lawyzheng/golru"
```

## 贡献说明

我们非常欢迎大家的贡献。如欲合并贡献，请遵循以下指引:
- 新建一个issue,并且叙述为什么这么做(解决一个bug，增加一个功能，等等)
- 根据核心团队对上述问题的反馈，提交一个PR，描述变更并链接到该问题。
- 新代码必须具有测试覆盖率。
- 如果代码是关于性能问题的，则必须在流程中包括基准测试(无论是在问题中还是在PR中)。
- 一般来说，我们希望`golru`尽可能简单。当你新建issue时请注意这一点。

## 鸣谢

感谢 [orcaman/concurrent-map](http://github.com/orcaman/concurrent-map) 基础思路。

## 许可证
MIT (详见 [LICENSE](https://github.com/LawyZheng/golru/blob/master/LICENSE) 文件)
