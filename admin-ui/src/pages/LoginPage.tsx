// admin-ui/src/pages/LoginPage.tsx
import React, { useState } from 'react'; // 导入 useState 用于管理加载状态
// 移除了未使用的 Typography 组件，保留 Form, Input, Button, Checkbox, message
import { Form, Input, Button, Checkbox, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useNavigate, Link as RouterLink } from 'react-router-dom';
import apiService from '../services/apiService'; // 导入 API Service
// 导入 useAuth Hook 和 AuthUser 类型。使用 `type AuthUser` 是因为 AuthUser 是一个 TypeScript 类型而非实际的 JavaScript 值，
// 这样有助于构建工具（如 Babel 或 esbuild）进行优化，确保类型信息在编译后被移除。
import { useAuth, type AuthUser } from '../contexts/AuthContext'; 
// 导入登录请求和响应的 DTO。使用 `type` 关键字指明这些是类型导入。
import type { UserLoginRes, UserLoginReq } from '../dto/auth_dto'; 

// const { Title } = Typography; // 如果需要页面内标题

/**
 * 登录表单提交的值的类型接口
 * 这些值直接来自 Ant Design Form 的字段
 */
interface LoginFormValues {
  usernameOrEmail: string; // 用户名或邮箱 (来自表单name属性)
  password: string;        // 密码 (来自表单name属性)
  remember?: boolean;       // “记住我”选项，可选
}

const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const auth = useAuth(); // 获取 AuthContext
  const [loading, setLoading] = useState(false); // 加载状态，防止重复提交

  // onFinish 函数现在使用 LoginFormValues 类型来注解表单提交的值
  const onFinish = async (values: LoginFormValues) => {
    setLoading(true); // 开始登录，设置加载状态为 true
    try {
      // 调用 API Service 的 post 方法进行登录
      // 后端期望的登录请求体是 dto.UserLoginReq: { username_or_email: string, password: string }
      // Ant Design Form 的 values 对象的键名是 Form.Item 的 name 属性值
      // loginData 明确指定类型为 UserLoginReq，确保了数据结构的正确性
      const loginData: UserLoginReq = { 
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
    } catch (error: unknown) { // 将 error 类型从 any 修改为 unknown，更符合 TypeScript 的类型安全实践
      // API 调用失败或发生其他错误
      // apiService 的响应拦截器通常会处理 HTTP 错误并显示消息
      console.error('登录请求失败:', error); // 仍然记录原始错误对象，便于调试

      // 根据错误类型提供更具体的反馈
      // 检查 error 是否是一个包含 response 属性的对象 (类似 AxiosError)
      if (typeof error === 'object' && error !== null && 'response' in error) {
        // 假设是类似 Axios 的错误结构，其中包含 response 对象
        // eslint-disable-next-line @typescript-eslint/no-explicit-any -- 暂时允许any类型以访问response上的未知属性,理想情况下应有更具体的错误类型
        const axiosError = error as any; 
        if (axiosError.response?.data?.msg) {
          // 如果 apiService 的拦截器没有显示消息，这里可以取消注释下一行来显示后端的错误消息
          // message.error(axiosError.response.data.msg); 
          console.error('详细错误信息 (来自后端):', axiosError.response.data.msg); // 记录后端提供的具体错误信息
        } else {
          // 如果 apiService 的拦截器没有显示消息，并且后端也没有提供具体的 msg
          // message.error('登录时发生网络或服务器错误。');
          console.error('登录时发生网络或服务器错误 (响应中无详细msg)。'); // 记录通用错误
        }
      } else if (error instanceof Error) {
        // 处理标准的 JavaScript Error 对象 (例如网络问题，或者在请求设置阶段抛出的错误)
        // 如果 apiService 的拦截器没有显示消息
        // message.error(error.message);
        console.error('错误信息 (Error实例):', error.message); // 记录错误消息
      } else {
        // 处理其他未知类型的错误
        // 如果 apiService 的拦截器没有显示消息
        // message.error('登录失败，发生未知错误。');
        console.error('登录失败，发生未知类型的错误。'); // 记录未知错误
      }
      // 注意：原代码注释提到 apiService 的响应拦截器已经处理了大部分 HTTP 错误并显示了 message。
      // 因此，上述的 message.error(...) 调用大多被注释掉了，以避免可能出现的重复错误提示。
      // 主要保留 console.error 用于开发和调试。
      // 如果实际场景中 apiService 的拦截器没有按预期显示消息，或者需要在此处覆盖其行为，
      // 可以取消相应 message.error(...) 行的注释。
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
