import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: { // 新增 server 配置块
    proxy: {
      // 代理所有以 /api/v1 开头的请求
      // 例如：前端请求 /api/v1/users 会被代理到 target + /api/v1/users
      // 即 http://localhost:8080/api/v1/users (假设 target 是 http://localhost:8080)

      // 字符串简写方式 (如果不需要太多自定义选项)：
      // '/api/v1': 'http://localhost:8080', // Vite 会自动处理路径拼接

      // 对象方式，提供更多配置选项：
      '/api/v1': {
        target: 'http://localhost:8080', // 后端 API 服务器的地址
        changeOrigin: true, // 必需：将请求头中的 Host 字段从开发服务器地址修改为目标服务器地址。
                            // 这对于后端依赖 Host 头进行路由或鉴权的 API 非常重要。
        
        // secure: false, // 如果你的后端 API 使用的是 HTTPS，并且证书是自签名的或无效的，
                         // 你可能需要设置此项为 false 来允许代理。默认为 true。

        // ws: true,      // 如果你需要代理 WebSocket 连接 (例如用于实时通讯功能)，
                         // 设置此项为 true。默认为 false。

        // rewrite: (path) => path.replace(/^\/api\/v1/, '/api/v1'), 
        // 路径重写规则。
        // 在当前配置下，Vite 的默认行为通常是正确的：
        // 前端请求 `/api/v1/some/endpoint`
        // 代理后的请求路径会是 `target` + `/api/v1/some/endpoint`
        // 即 `http://localhost:8080/api/v1/some/endpoint`
        // 只有当后端期望的路径与前端请求的路径前缀不完全一致时，才需要显式重写。
        // 例如，如果后端实际路径是 `/actual-api/v1/...` 而不是 `/api/v1/...`，
        // 且 target 是 `http://localhost:8080`，你可能需要：
        // rewrite: (path) => path.replace(/^\/api\/v1/, '/actual-api/v1')
        // 但在当前场景下，我们希望保留 /api/v1 前缀，所以不需要显式 rewrite。
      },
      // 如果有其他需要代理的 API 前缀，可以在此继续添加
      // 例如，代理 /api/v2 到另一个后端服务：
      // '/api/v2': {
      //   target: 'http://localhost:8081', // 另一个后端服务地址
      //   changeOrigin: true,
      // },
    },
  },
})
