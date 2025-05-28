// admin-ui/src/dto/auth_dto.ts

/**
 * 用户登录请求的数据传输对象 (DTO)
 * 对应后端期望的请求体结构
 */
export interface UserLoginReq {
  username_or_email: string; // 用户名或邮箱
  password: string;          // 密码
}

/**
 * 用户登录响应的数据传输对象 (DTO)
 * 对应后端 /users/login 接口成功时返回的 data 字段结构
 */
export interface UserLoginRes {
  access_token: string; // JWT 访问令牌
  user_id: string;      // 用户ID (请根据后端实际返回类型调整，可能是 number)
  username: string;     // 用户名
  role?: string;        // 用户角色，可选
}
