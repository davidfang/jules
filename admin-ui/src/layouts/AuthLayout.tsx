// admin-ui/src/layouts/AuthLayout.tsx
import React from 'react';
import { Outlet } from 'react-router-dom';
import { Layout, Row, Col, Typography } from 'antd';
const { Title } = Typography;

const AuthLayout: React.FC = () => (
  <Layout style={{ minHeight: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center', background: '#f0f2f5' }}>
    <Row justify="center" align="middle" style={{width: '100%'}}>
      <Col xs={20} sm={16} md={12} lg={8} xl={6} style={{ padding: '20px', background: '#fff', borderRadius: '8px', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.15)' }}>
        <div style={{textAlign: 'center', marginBottom: '24px'}}>
           <Title level={2}>系统登录</Title> {/* TODO: 根据路由动态显示标题 */}
        </div>
        <Outlet />
      </Col>
    </Row>
  </Layout>
);
export default AuthLayout;
