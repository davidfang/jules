// admin-ui/src/pages/LoginPage.tsx
import React, { useState } from 'react'; // 导入 useState 用于管理加载状态
import { Form, Input, Button, Checkbox, Typography, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useNavigate, Link as RouterLink } from 'react-router-dom';
import apiService from '../services/apiService'; // 导入 API Service
import { useAuth, AuthUser } from '../contexts/AuthContext'; // 导入 useAuth Hook 和 AuthUser 类型
import { UserLoginRes } from '../dto/auth_dto'; // 导入登录响应 DTO

// const { Title } = Typography; // 如果需要页面内标题

const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const auth = useAuth(); // 获取 AuthContext
  const [loading, setLoading] = useState(false); // 加载状态，防止重复提交

  const onFinish = async (values: any) => {
    setLoading(true); // 开始登录，设置加载状态为 true
    try {
      // 调用 API Service 的 post 方法进行登录
      // 后端期望的登录请求体是 dto.UserLoginReq: { username_or_email: string, password: string }
      // Ant Design Form 的 values 对象的键名是 Form.Item 的 name 属性值
      const loginData = {
        username_or_email: values.usernameOrEmail, // 确保与 Form.Item name="usernameOrEmail" 一致
        password: values.password,
      };
      
      // loginRes 的类型应与后端 /users/login 端点返回的 JSON 结构匹配，
      // 并且与 dto.UserLoginRes 结构体兼容。
      // apiService.post 会自动处理 response.data，所以 loginRes 直接是后端 data 字段的内容。
      const loginRes: UserLoginRes = await apiService.post<UserLoginRes>('/users/login', loginData);

      if (loginRes && loginRes.access_token && loginRes.user_id && loginRes.username) {
        // 登录成功，从响应数据中获取 token 和用户信息
        const userData: AuthUser = {
          id: loginRes.user_id,
          username: loginRes.username,
          role: loginRes.role || 'role_user', // 假设后端会返回 role，否则提供默认值
        };
        
        // 调用 AuthContext 的 login 方法保存认证信息
        await auth.login(loginRes.access_token, userData);
        
        message.success('登录成功！');
        navigate('/'); // 跳转到后台首页 (或其他受保护的默认页)
      } else {
        // 响应数据不符合预期
        message.error('登录失败：无效的响应数据。');
        console.error('Login response data is invalid:', loginRes);
      }
    } catch (error: any) {
      // API 调用失败或发生其他错误
      // apiService 的响应拦截器已经处理了大部分 HTTP 错误并显示了 message
      // 这里可以根据需要添加额外的错误处理逻辑，或者依赖拦截器的提示
      console.error('登录请求失败:', error);
      // message.error(error.response?.data?.msg || error.message || '登录失败，请稍后重试。'); // 如果拦截器没有reject(error.response.data)
    } finally {
      setLoading(false); // 结束登录，设置加载状态为 false
    }
  };

  return (
    <>
      <Form
        name="normal_login"
        initialValues={{ remember: true }}
        onFinish={onFinish}
        layout="vertical"
      >
        <Form.Item
          name="usernameOrEmail" // 与上面 loginData 中使用的键名一致
          label="用户名或邮箱"
          rules={[{ required: true, message: '请输入您的用户名或邮箱!' }]}
        >
          <Input prefix={<UserOutlined />} placeholder="请输入用户名或邮箱" size="large" disabled={loading} />
        </Form.Item>
        <Form.Item
          name="password"
          label="密码"
          rules={[{ required: true, message: '请输入您的密码!' }]}
        >
          <Input.Password prefix={<LockOutlined />} placeholder="请输入密码" size="large" disabled={loading} />
        </Form.Item>
        <Form.Item>
          <Form.Item name="remember" valuePropName="checked" noStyle>
            <Checkbox disabled={loading}>记住我</Checkbox>
          </Form.Item>
          <RouterLink style={{ float: 'right' }} to="/auth/forgot-password"> 
            忘记密码?
          </RouterLink>
        </Form.Item>
        <Form.Item>
          <Button type="primary" htmlType="submit" style={{ width: '100%' }} size="large" loading={loading}>
            登录
          </Button>
        </Form.Item>
        <div style={{ textAlign: 'center' }}>
          还没有账户? <RouterLink to="/auth/register">现在注册!</RouterLink>
        </div>
      </Form>
    </>
  );
};
export default LoginPage;
