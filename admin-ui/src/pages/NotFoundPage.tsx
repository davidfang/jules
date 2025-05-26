// admin-ui/src/pages/NotFoundPage.tsx
import React from 'react';
import { Result, Button } from 'antd';
import { Link } from 'react-router-dom';

const NotFoundPage: React.FC = () => (
  <Result
    status="404"
    title="404 - 页面未找到"
    subTitle="抱歉，您访问的页面不存在或已被移除。"
    extra={
      <Button type="primary">
        <Link to="/">返回首页</Link>
      </Button>
    }
    style={{ 
      // 可以添加一些样式使页面在 AuthLayout 或 AdminLayout 中都看起来合适
      // 如果在 AuthLayout 中，它已经居中了。
      // 如果可能在 AdminLayout 中（例如，用户在登录后访问了一个不存在的后台路径），
      // 可能需要确保它在内容区域内良好显示。
      // 当前 Result 组件通常能自适应。
    }}
  />
);

export default NotFoundPage;
