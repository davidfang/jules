// admin-ui/src/pages/RegisterPage.tsx
import React, { useState } from 'react';
import { Form, Input, Button, Typography, message } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined } from '@ant-design/icons';
import { Link, useNavigate } from 'react-router-dom';
import apiService from '../services/apiService'; // 确保 apiService 路径正确
// 导入用户注册请求的 DTO 类型
import type { UserRegisterReq } from '../dto/auth_dto'; // 路径可能需要调整

const { Title } = Typography;

/**
 * 注册表单提交的值的类型接口。
 * 包含确认密码字段，用于表单校验。
 */
interface RegisterFormValues {
  username: string;        // 用户名
  email: string;           // 邮箱地址
  password: string;        // 密码
  confirmPassword?: string; // 确认密码字段，在发送给后端前会移除
}

const RegisterPage: React.FC = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm(); // 用于表单实例，例如重置表单

  // onFinish 函数现在使用 RegisterFormValues 类型来注解表单提交的值
  const onFinish = async (values: RegisterFormValues) => { 
    setLoading(true);
    // 确认密码校验逻辑已在 Form.Item rules 中处理
    // requestData 明确指定类型为 UserRegisterReq，确保了发送到后端的数据结构的正确性
    const requestData: UserRegisterReq = {
      username: values.username,
      email: values.email,
      password: values.password,
    };

    // 根据 UserRegisterReq DTO 构造请求数据 (已在上一行完成)

    try {
      // 调用 API Service 的 post 方法进行用户注册
      // 后端成功注册通常返回创建的用户信息 (不含密码) 或简单的成功状态
      // 这里我们假设后端成功时返回的数据结构可以忽略，主要关注是否成功调用
      await apiService.post('/users/register', requestData); // 注意：apiService 的 baseURL 会自动处理 API 前缀 (如 /api/v1)
                                                          // 此处路径是相对于 baseURL 的。

      message.success('注册成功！您现在可以登录了。');
      form.resetFields(); // 清空表单字段
      navigate('/auth/login'); // 导航到登录页面

    } catch (error: unknown) { // 将 error 类型设置为 unknown 以进行更安全的处理
      console.error('注册失败:', error); // 在控制台记录完整错误信息
      let errorMessage = '注册失败，请稍后重试。'; // 默认错误消息

      // 尝试从 Axios 错误结构中提取后端返回的错误消息
      // apiService 拦截器可能已经处理并显示了部分错误消息，
      // 但如果需要更细致的控制或后端消息未被拦截器捕获，可以在此处理。
      if (typeof error === 'object' && error !== null && 'response' in error) {
        // 使用更安全的类型断言来访问 response.data.msg
        const axiosError = error as { response?: { data?: { msg?: string } } }; 
        if (axiosError.response?.data?.msg) {
          errorMessage = axiosError.response.data.msg;
        }
      }
      // 如果错误是 Error 实例，且没有更具体的后端消息，可以使用其 message 属性
      // (但通常 Axios 错误中的 response.data.msg 更具体，且可能已被拦截器处理)
      // else if (error instanceof Error) {
      //   errorMessage = error.message;
      // }
      
      message.error(errorMessage); // 显示最终确定的错误消息
    } finally {
      setLoading(false); // 无论成功或失败，结束加载状态
    }
  };

  return (
    <div style={{ maxWidth: 400, margin: 'auto', paddingTop: 50 }}>
      <Title level={2} style={{ textAlign: 'center', marginBottom: 24 }}>
        注册新账户
      </Title>
      <Form
        form={form}
        name="register"
        onFinish={onFinish}
        layout="vertical"
        initialValues={{ remember: true }}
        scrollToFirstError
      >
        <Form.Item
          name="username"
          label="用户名"
          rules={[
            { required: true, message: '请输入您的用户名!' },
            { min: 3, message: '用户名至少需要3个字符!' },
            { max: 50, message: '用户名不能超过50个字符!' },
          ]}
        >
          <Input prefix={<UserOutlined />} placeholder="请输入用户名" size="large" />
        </Form.Item>

        <Form.Item
          name="email"
          label="邮箱地址"
          rules={[
            { required: true, message: '请输入您的邮箱地址!' },
            { type: 'email', message: '请输入有效的邮箱地址!' },
          ]}
        >
          <Input prefix={<MailOutlined />} placeholder="请输入邮箱地址" size="large" />
        </Form.Item>

        <Form.Item
          name="password"
          label="密码"
          rules={[
            { required: true, message: '请输入您的密码!' },
            { min: 6, message: '密码至少需要6个字符!' },
            // 可以添加更复杂的密码强度校验规则
          ]}
          hasFeedback // 在输入时显示校验状态图标
        >
          <Input.Password prefix={<LockOutlined />} placeholder="请输入密码" size="large" />
        </Form.Item>

        <Form.Item
          name="confirmPassword"
          label="确认密码"
          dependencies={['password']} // 依赖于 password 字段
          hasFeedback
          rules={[
            { required: true, message: '请再次输入您的密码以确认!' },
            ({ getFieldValue }) => ({
              validator(_, value) {
                if (!value || getFieldValue('password') === value) {
                  return Promise.resolve();
                }
                return Promise.reject(new Error('两次输入的密码不一致!'));
              },
            }),
          ]}
        >
          <Input.Password prefix={<LockOutlined />} placeholder="请确认密码" size="large" />
        </Form.Item>

        <Form.Item>
          <Button type="primary" htmlType="submit" style={{ width: '100%' }} loading={loading} size="large">
            注册
          </Button>
        </Form.Item>

        <div style={{ textAlign: 'center' }}>
          已经有账户了? <Link to="/auth/login">返回登录</Link>
        </div>
      </Form>
    </div>
  );
};

export default RegisterPage;
