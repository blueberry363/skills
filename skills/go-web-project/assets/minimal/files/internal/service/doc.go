// Package service 是 MVC 的 Service 层：业务规则与事务边界。
//
// 骨架阶段本包为空。新增资源的标准动作：
//  1. model：定义模型与 store 接口及其 GORM 实现；
//  2. service：新建 <resource>.go，定义 Service 接口与实现，事务在此开启；
//  3. controller：新建 <resource>.go，绑定参数并调用 Service；
//  4. router：在 New 里加一行 Register。
//
// 约束见仓库根目录 AGENTS.md 第 3 段：本层禁止 import gin 与 net/http。
package service
