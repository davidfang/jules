// admin-ui/src/contexts/AuthContext.tsx
import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import apiService from '../services/apiService'; // 导入 API Service
import { Spin, message } from 'antd'; // Spin 用于加载指示器

// 1. 定义 AuthUser 接口 (当前登录用户的基本信息)
export interface AuthUser {
  id: number; // 或 uint，与后端模型一致
  username: string;
  role: string; // 用户角色，例如 'admin', 'user'
  // 可以根据需要添加更多字段，例如 email, avatar 等
}

// 2. 定义 AuthContextType 接口 (Context 提供的值和方法)
export interface AuthContextType {
  isAuthenticated: boolean;        // 用户是否已认证
  user: AuthUser | null;           // 当前认证的用户信息，如果未认证则为 null
  token: string | null;            // JWT 认证 Token
  isLoading: boolean;              // 是否正在加载初始认证状态 (例如，从 localStorage 恢复并验证 Token)
  login: (newToken: string, userData: AuthUser) => Promise<void>; // 登录方法
  logout: () => void;              // 登出方法
  // 可以添加其他方法，例如 refreshToken 等
}

// 创建 AuthContext，初始值为 undefined，因为 Provider 会提供实际值
const AuthContext = createContext<AuthContextType | undefined>(undefined);

// 3. 实现 AuthProvider 组件
interface AuthProviderProps {
  children: ReactNode; // 子组件
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [user, setUser] = useState<AuthUser | null>(null);
  const [token, setToken] = useState<string | null>(localStorage.getItem('authToken')); // 从 localStorage 初始化 token
  const [isLoading, setIsLoading] = useState<boolean>(true); // 初始时为 true，表示正在加载状态

  // useEffect Hook: 应用加载时执行，用于恢复和验证认证状态
  useEffect(() => {
    const initializeAuth = async () => {
      const storedToken = localStorage.getItem('authToken');
      if (storedToken) {
        setToken(storedToken); // 设置 token 到 state
        // apiService 拦截器会自动添加 token 到请求头
        try {
          // 尝试调用受保护的 /me/profile 端点获取用户信息
          // 后端返回的 model.User 结构需要与 AuthUser 接口兼容或进行转换
          const profileData: AuthUser = await apiService.get<AuthUser>('/me/profile');
          if (profileData && profileData.id) {
            setUser(profileData);
            setIsAuthenticated(true);
            message.success(`欢迎回来, ${profileData.username}!`);
          } else {
            // Token 有效但获取用户信息失败，或返回数据不符合预期
            throw new Error('获取用户信息失败或用户数据无效');
          }
        } catch (error) {
          // Token 无效 (例如过期) 或网络错误
          console.error('恢复登录状态失败:', error);
          localStorage.removeItem('authToken'); // 清除无效的 token
          setToken(null);
          setUser(null);
          setIsAuthenticated(false);
          // message.warn('会话已过期或无效，请重新登录。'); // 可选提示
        }
      }
      setIsLoading(false); // 无论结果如何，加载过程结束
    };

    initializeAuth();
  }, []); // 空依赖数组表示仅在组件挂载时执行一次

  // login 方法：保存 Token 和用户信息，更新认证状态
  const login = async (newToken: string, userData: AuthUser) => {
    localStorage.setItem('authToken', newToken); // 将 Token 保存到 localStorage
    setToken(newToken);
    setUser(userData);
    setIsAuthenticated(true);
    // 通常在 LoginPage 中处理导航和成功消息
  };

  // logout 方法：清除 Token 和用户信息，更新认证状态
  const logout = () => {
    localStorage.removeItem('authToken'); // 从 localStorage 清除 Token
    setToken(null);
    setUser(null);
    setIsAuthenticated(false);
    // 通常在 AdminLayout 中处理导航和成功消息
    // message.success('已成功退出登录。'); // 移至调用处
    // window.location.href = '/login'; // 或者使用 navigate
  };

  // 如果正在加载初始状态，则显示全局加载指示器
  if (isLoading) {
    return (
      // 使用 Spin 的 fullscreen 属性（如果 antd 版本支持）来实现全屏加载效果，
      // 这通常能让 tip 属性按预期工作，解决相关警告。
      // 如果 fullscreen 属性不可用或导致其他问题，则应考虑移除 tip 属性。
      // 外部的 div 在 Spin 使用 fullscreen 时可能不再是完全必要的，
      // 但暂时保留它，以确保在 Spin 的 fullscreen 实现不包含居中逻辑时，内容仍能居中。
      // 如果 Spin fullscreen 本身就能完美居中，则此外部 div 可以被移除以简化结构。
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large" tip="正在加载应用状态..." fullscreen />
      </div>
    );
  }

  // 提供 Context 值给子组件
  return (
    <AuthContext.Provider value={{ isAuthenticated, user, token, login, logout, isLoading }}>
      {children}
    </AuthContext.Provider>
  );
};

// 4. 导出 useAuth 自定义 Hook
// useAuth Hook 简化了在组件中访问 AuthContext 的方式。
// 它确保了 Context 在被消费时是已定义的。
export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth 必须在 AuthProvider 内部使用');
  }
  return context;
};
