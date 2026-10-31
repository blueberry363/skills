// Package reqctx 定义 gin.Context 的键名，供 middleware 与 response 共用。
//
// middleware 依赖 response（错误体），response 不能反向依赖 middleware，
// 因此键名单列一包，避免两边各写一份字面量后静默漂移。
package reqctx

// RequestIDKey 是 request_id 在 gin.Context 中的键名。
const RequestIDKey = "request_id"
