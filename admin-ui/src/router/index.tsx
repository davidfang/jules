// admin-ui/src/router/index.tsx
// 导入 createBrowserRouter 函数用于创建路由实例
// RouteObject 作为类型导入，以明确其仅为类型注解，有助于 Vite 等构建工具处理
import { createBrowserRouter, type RouteObject } from 'react-router-dom'; // 移除了 Navigate，因为 ProtectedRoute 内部处理
import AdminLayout from '../layouts/AdminLayout';
import AuthLayout from '../layouts/AuthLayout';
import LoginPage from '../pages/LoginPage';
import DashboardPage from '../pages/DashboardPage';
import NotFoundPage from '../pages/NotFoundPage';
import ProtectedRoute from './ProtectedRoute'; // 导入 ProtectedRoute

// 移除了旧的 isAuthenticated 和 ProtectedRoute 组件定义，因为它们现在在 ProtectedRoute.tsx 中

const routes: RouteObject[] = [
  {
    path: '/auth', // 统一的认证相关路由前缀
    element: <AuthLayout />,
    children: [
      { path: 'login', element: <LoginPage /> },
      // 可以添加注册页面路由等
      // { path: 'register', element: <RegisterPage /> },
      // { path: 'forgot-password', element: <ForgotPasswordPage /> },
    ],
  },
  {
    // 针对 /login 的单独路由，避免被 ProtectedRoute 包装
    // 如果用户已认证并尝试访问 /login，可以考虑在 LoginPage 内部处理重定向
    path: '/login',
    element: <AuthLayout />,
    children: [{ index: true, element: <LoginPage /> }],
  },
  {
    // 受保护的路由组，使用 ProtectedRoute 作为 element
    // ProtectedRoute 内部会渲染 <Outlet />，所以 AdminLayout 及其子路由会作为其子元素渲染
    element: <ProtectedRoute />, 
    children: [
      {
        path: '/', // AdminLayout 作为此受保护路径下的布局
        element: <AdminLayout />,
        children: [
          { index: true, element: <DashboardPage /> }, // 默认首页 /
          { path: 'dashboard', element: <DashboardPage /> }, // /dashboard
          // 示例：用户管理页面路由 (后续创建 UsersPage.tsx)
          // { path: 'users', element: <UsersPage /> }, 
          // 更多后台管理页面可以在此添加
        ],
      },
    ],
  },
  {
    path: '*', // 匹配所有未定义的路由，作为 404 页面
    element: <NotFoundPage />,
  },
];

const router = createBrowserRouter(routes);
export default router;
