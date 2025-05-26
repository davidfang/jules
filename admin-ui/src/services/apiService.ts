// admin-ui/src/services/apiService.ts
import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import { message } from 'antd'; // 用于全局错误提示

// 1. 创建 Axios 实例
const apiService: AxiosInstance = axios.create({
  // 2. 设置基础 URL
  // Vite 环境变量通过 import.meta.env 访问
  // VITE_API_BASE_URL 应在 .env 文件中定义，例如 VITE_API_BASE_URL=http://localhost:8080/api/v1
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1', // 提供一个默认值以防环境变量未设置
  timeout: 10000, // 请求超时时间 (ms)
  headers: {
    'Content-Type': 'application/json',
  },
});

// 3. 请求拦截器
apiService.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // 从 localStorage 获取认证 Token
    const token = localStorage.getItem('authToken');
    if (token) {
      // 如果 Token 存在，则将其添加到 Authorization 请求头中
      // 格式为 "Bearer <token>"
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    // 对请求错误做些什么
    // 例如，可以在这里记录错误日志
    console.error('Axios Request Interceptor Error:', error);
    return Promise.reject(error);
  }
);

// 4. 响应拦截器
apiService.interceptors.response.use(
  (response: AxiosResponse) => {
    // 对响应数据做点什么
    // 通常，后端API会有一个统一的响应结构，例如 { code: 0, data: {}, msg: "" }
    // 这里我们直接返回 response.data，假设它已经是我们期望的格式
    // 或者，可以在这里进一步处理后端返回的业务错误码
    // 例如：
    // if (response.data && response.data.code !== 0) {
    //   message.error(response.data.msg || '操作失败');
    //   return Promise.reject(new Error(response.data.msg || 'Error'));
    // }
    return response.data; // 直接返回后端响应的 data 部分
  },
  (error) => {
    // 对响应错误做点什么
    console.error('Axios Response Interceptor Error:', error);

    if (error.response) {
      // 请求已发出，服务器以状态码响应
      const { status, data } = error.response;
      switch (status) {
        case 400:
          // 通常是参数校验错误或业务逻辑错误
          message.error(data?.msg || '请求参数错误');
          break;
        case 401:
          // 未授权错误
          message.error(data?.msg || '认证失败或 Token 已过期，请重新登录。');
          // 清除本地存储的 Token
          localStorage.removeItem('authToken');
          // 清除用户状态 (如果使用 Context 或 Redux)
          // authContext.logout(); // 假设有 authContext
          // 重定向到登录页面
          // 使用 window.location.href 进行硬跳转。
          // 如果在 React 组件内部，可以使用 useNavigate()。
          // 注意：在非组件文件中直接使用 React Router 的 navigate 可能比较复杂，
          // window.location.href 是一个简单直接的方法，但会导致页面刷新。
          // 更好的做法是在应用层面处理重定向，例如通过 AuthContext。
          if (window.location.pathname !== '/login' && window.location.pathname !== '/auth/login') {
            window.location.href = '/login'; 
          }
          break;
        case 403:
          message.error(data?.msg || '您没有权限执行此操作。');
          break;
        case 404:
          message.error(data?.msg || '请求的资源未找到。');
          break;
        case 500:
        case 502:
        case 503:
        case 504:
          message.error(data?.msg || '服务器发生错误，请稍后重试。');
          break;
        default:
          message.error(data?.msg || `发生未知错误 (状态码: ${status})`);
      }
    } else if (error.request) {
      // 请求已发出，但没有收到响应 (例如网络错误)
      message.error('网络请求失败，请检查您的网络连接。');
    } else {
      // 在设置请求时发生了一些事情，触发了一个错误
      message.error(`请求配置错误: ${error.message}`);
    }
    return Promise.reject(error); // 将错误继续传递下去，以便调用方可以捕获和处理
  }
);

// 5. 封装 HTTP 方法
// 提供类型化的 Promise 返回值，T 是期望的响应数据类型
const httpRequest = {
  get: <T = any>(url: string, config?: AxiosRequestConfig): Promise<T> => 
    apiService.get<T, T>(url, config), // AxiosResponse<T> 被拦截器处理为 T

  post: <T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> =>
    apiService.post<T, T>(url, data, config),

  put: <T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> =>
    apiService.put<T, T>(url, data, config),

  delete: <T = any>(url: string, config?: AxiosRequestConfig): Promise<T> =>
    apiService.delete<T, T>(url, config),
  
  // 可以添加其他方法，例如 patch 等
  // patch: <T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> =>
  //   apiService.patch<T, T>(url, data, config),
};

export default httpRequest;
