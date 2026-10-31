// Package model 是 MVC 的 Model 层：数据模型 + 数据访问。
//
// 骨架阶段本包为空。新增资源的标准动作：
//  1. 新建 <resource>.go 定义模型，字段与 db/migrations 中的表保持一致；
//  2. 新建 <resource>_store.go 定义 Store 接口与 GORM 实现；
//  3. 所有查询带 context（db.WithContext），显式 Select 字段。
//
// 约束见仓库根目录 AGENTS.md 第 6 段：禁止 AutoMigrate，schema 以迁移文件为唯一来源。
package model
