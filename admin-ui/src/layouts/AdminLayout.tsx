// admin-ui/src/layouts/AdminLayout.tsx
import React, { useState } from 'react';
import { Outlet, Link, useNavigate } from 'react-router-dom';
import { Layout, Menu, Breadcrumb, Avatar, Dropdown, Space, message } from 'antd'; // 导入 message
import {
  DesktopOutlined,
  PieChartOutlined,
  UserOutlined,
  LogoutOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { useAuth } from '../contexts/AuthContext'; // 导入 useAuth Hook

const { Header, Content, Footer, Sider } = Layout;

type MenuItem = Required<MenuProps>['items'][number];

function getItem(
  label: React.ReactNode,
  key: React.Key,
  icon?: React.ReactNode,
  children?: MenuItem[],
): MenuItem {
  return {
    key,
    icon,
    children,
    label,
  } as MenuItem;
}

const items: MenuItem[] = [
  getItem(<Link to="/">首页</Link>, '1', <PieChartOutlined />),
  getItem(<Link to="/users">用户管理</Link>, '2', <DesktopOutlined />), 
];

const AdminLayout: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const { user, isAuthenticated, logout } = useAuth(); // 使用 useAuth 获取用户状态和登出方法

  const handleMenuClick: MenuProps['onClick'] = (e) => {
    if (e.key === 'logout') {
      logout(); // 调用 AuthContext 的 logout 方法
      message.success('已成功退出登录。');
      navigate('/login'); 
    } else if (e.key === 'profile') {
      // navigate('/profile'); // 假设未来有 /profile 路由
      console.log('跳转到个人资料页 (TODO)');
    }
  };

  const menuItems: MenuProps['items'] = [
    { key: 'profile', icon: <UserOutlined />, label: '个人中心' },
    { key: 'settings', icon: <SettingOutlined />, label: '账户设置' },
    { type: 'divider' },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录' },
  ];


  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider collapsible collapsed={collapsed} onCollapse={(value) => setCollapsed(value)}>
        <div style={{ height: 32, margin: 16, background: 'rgba(255, 255, 255, 0.2)', textAlign: 'center', lineHeight: '32px', color: 'white', fontWeight: 'bold' }}>
          {collapsed ? 'GS' : 'Go System'}
        </div>
        <Menu theme="dark" defaultSelectedKeys={['1']} mode="inline" items={items} />
      </Sider>
      <Layout>
        <Header style={{ padding: '0 16px', background: '#fff', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Breadcrumb style={{ margin: '0' }}>
            <Breadcrumb.Item>首页</Breadcrumb.Item>
          </Breadcrumb>
          {isAuthenticated && user ? ( // 仅当用户已认证且用户信息存在时显示下拉菜单
            <Dropdown menu={{ items: menuItems, onClick: handleMenuClick }}>
              <a onClick={(e) => e.preventDefault()} style={{cursor: 'pointer'}}>
                <Space>
                  <Avatar size="small" icon={<UserOutlined />} />
                  {user.username} {/* 显示认证用户的用户名 */}
                </Space>
              </a>
            </Dropdown>
          ) : (
            // （可选）用户未认证或用户信息加载中时的占位符或链接
            <Link to="/login">登录</Link>
          )}
        </Header>
        <Content style={{ margin: '16px' }}>
          <div style={{ padding: 24, minHeight: 360, background: '#fff' }}> 
            <Outlet />
          </div>
        </Content>
        <Footer style={{ textAlign: 'center' }}>
          Go Base System ©{new Date().getFullYear()} Created by YourName/Company
        </Footer>
      </Layout>
    </Layout>
  );
};
export default AdminLayout;
