// admin-ui/src/router/ProtectedRoute.tsx
import React from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { Spin } from 'antd';

const ProtectedRoute: React.FC = () => {
  const { isAuthenticated, isLoading } = useAuth();
  const location = useLocation(); // 获取当前位置，用于登录后重定向

  // 1. 如果正在加载初始认证状态，则显示全局加载指示器
  if (isLoading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <Spin size="large" tip="正在验证身份..." />
      </div>
    );
  }

  // 2. 如果用户未认证且加载已完成，则重定向到登录页
  if (!isAuthenticated) {
    // 将用户尝试访问的原始路径 (location.pathname) 作为 state 传递给登录页。
    // 登录成功后，LoginPage 可以使用此 state 将用户重定向回他们最初想访问的页面。
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // 3. 如果用户已认证，则渲染子路由 (通过 <Outlet />)
  // 子路由通常是 AdminLayout 或其他受保护的布局/页面。
  return <Outlet />;
};

export default ProtectedRoute;
