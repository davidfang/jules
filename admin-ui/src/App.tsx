// admin-ui/src/App.tsx
import { RouterProvider } from 'react-router-dom';
import router from './router'; // 导入在 router/index.tsx 中创建的 router
import { ConfigProvider, App as AntApp } from 'antd'; // 导入 Ant Design 的 ConfigProvider 和 App 组件
import zhCN from 'antd/locale/zh_CN'; // 引入中文语言包
import 'dayjs/locale/zh-cn'; // 确保 dayjs 的中文语言包也被加载 (Ant Design 依赖 dayjs)
import dayjs from 'dayjs';
import { AuthProvider } from './contexts/AuthContext'; // 导入 AuthProvider

// 设置 dayjs 的全局 locale
dayjs.locale('zh-cn');

function App() {
  return (
    // ConfigProvider 用于 Ant Design 的全局配置，例如国际化。
    // locale={zhCN} 将 Ant Design 组件的默认语言设置为中文。
    <ConfigProvider locale={zhCN}>
      {/* AntApp 组件用于启用 Ant Design 的全局特性，如 message, notification, modal 等的上下文消费。 */}
      <AntApp>
        {/* AuthProvider 包裹 RouterProvider，使得整个应用的路由和组件都能访问 AuthContext。 */}
        <AuthProvider>
          {/* RouterProvider 负责提供路由上下文给应用中的所有路由组件。 */}
          <RouterProvider router={router} />
        </AuthProvider>
      </AntApp>
    </ConfigProvider>
  );
}

export default App;
