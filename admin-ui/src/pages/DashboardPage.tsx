// admin-ui/src/pages/DashboardPage.tsx
import React from 'react';
import { Typography, Card, Row, Col, Statistic } from 'antd';
import { ArrowUpOutlined, ArrowDownOutlined, UserOutlined, ShoppingCartOutlined, LineChartOutlined } from '@ant-design/icons';

const { Title, Paragraph } = Typography;

const DashboardPage: React.FC = () => {
  return (
    <div>
      <Title level={2} style={{ marginBottom: '24px' }}>欢迎来到管理后台</Title>
      <Paragraph>
        这是仪表盘页面，您可以在这里快速了解系统的核心指标和最新动态。
      </Paragraph>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={8} lg={8} xl={6}>
          <Card>
            <Statistic
              title="活跃用户"
              value={1128}
              precision={0}
              valueStyle={{ color: '#3f8600' }}
              prefix={<UserOutlined />}
              suffix="人"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={8} xl={6}>
          <Card>
            <Statistic
              title="本月订单"
              value={93}
              precision={0}
              valueStyle={{ color: '#cf1322' }}
              prefix={<ShoppingCartOutlined />}
              suffix="单"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={8} xl={6}>
          <Card>
            <Statistic
              title="访问量 (今日)"
              value={11.28}
              precision={2}
              valueStyle={{ color: '#3f8600' }}
              prefix={<LineChartOutlined />}
              suffix="%"
            />
             {/* 示例：与昨日比较的箭头 */}
            <div style={{marginTop: 10, fontSize: 12, color: 'gray'}}>
                <ArrowUpOutlined style={{color: '#3f8600'}} /> <span>较昨日上升 2.5%</span>
            </div>
          </Card>
        </Col>
         <Col xs={24} sm={12} md={8} lg={8} xl={6}>
          <Card>
            <Statistic
              title="待处理任务"
              value={5}
              valueStyle={{ color: '#d48806' }}
              prefix={<UserOutlined />} // 可以替换为更合适的图标
            />
             <div style={{marginTop: 10, fontSize: 12, color: 'gray'}}>
                <ArrowDownOutlined style={{color: '#cf1322'}} /> <span>较上周减少 1</span>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 后续可以添加更多图表和数据展示区域 */}
      {/* 例如： */}
      {/* <Row gutter={[16, 16]} style={{ marginTop: '24px' }}> */}
      {/*   <Col span={12}> */}
      {/*     <Card title="用户增长趋势"> */}
      {/*       <Paragraph>这里可以放置一个用户增长的折线图。</Paragraph> */}
      {/*     </Card> */}
      {/*   </Col> */}
      {/*   <Col span={12}> */}
      {/*     <Card title="订单分布"> */}
      {/*       <Paragraph>这里可以放置一个订单类型的饼图。</Paragraph> */}
      {/*     </Card> */}
      {/*   </Col> */}
      {/* </Row> */}
    </div>
  );
};

export default DashboardPage;
