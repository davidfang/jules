import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App.tsx';
import 'antd/dist/reset.css'; // Ant Design v5+ 使用 reset.css
import './index.css'; // 保留项目默认的全局样式文件 (如果需要)

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
