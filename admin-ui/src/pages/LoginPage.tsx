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
    setLoading(true); // setLoading(true) 应该在 try 块之外，或者在 try 块的最开始
                     // 但 setLoading(false) 必须在 finally 中以确保执行。
                     // 为了代码清晰，我们将 setLoading(true) 放在 try 之前。
    
    // 构造登录请求数据，确保与 UserLoginReq DTO 结构一致
    const loginData: UserLoginReq = { 
      username_or_email: values.usernameOrEmail,
      password: values.password,
    };
      
    try {
      // 定义期望的后端完整响应结构
      interface BackendLoginResponse {
        code: number;       // 业务响应码，例如 0 表示成功
        msg: string;        // 响应消息
        data: UserLoginRes; // 实际的业务数据，UserLoginRes 包含 access_token 等
      }
      
      // 调用 API Service 发送登录请求
      // backendResponse 现在是后端返回的完整响应体 {code, msg, data: UserLoginRes}
      const backendResponse = await apiService.post<BackendLoginResponse>('/users/login', loginData);

      // 检查业务响应码是否表示成功 (通常 code === 0) 且 data 字段存在
      if (backendResponse && backendResponse.code === 0 && backendResponse.data) {
        const loginBusinessData = backendResponse.data; // loginBusinessData 是 UserLoginRes 类型

        // 校验从后端获取的关键业务数据是否存在
        if (loginBusinessData.access_token && loginBusinessData.user_id && loginBusinessData.username) {
          // 构造 AuthUser 对象用于 AuthContext
          const userData: AuthUser = {
            id: loginBusinessData.user_id,
            username: loginBusinessData.username,
            role: loginBusinessData.role || 'role_user', // 提供默认角色以防后端未返回
          };
          
          // 调用 AuthContext 的 login 方法更新应用认证状态
          await auth.login(loginBusinessData.access_token, userData);
          
          message.success('登录成功！'); // 用户提示：登录成功
          navigate('/'); // 导航到应用首页或仪表盘
        } else {
          // 虽然业务码表示成功，但响应的 data 字段中缺少必要的字段
          console.error('登录成功但响应数据不完整:', backendResponse.data);
          message.error('登录失败：服务器返回的用户信息不完整。');
        }
      } else {
        // 业务码表示失败 (code !== 0)，或响应结构不符合预期 (例如 backendResponse 为 null)
        console.error('登录失败，后端业务码非0或响应格式问题:', backendResponse);
        // 优先使用后端返回的 msg，如果不存在则提供通用错误信息
        message.error(backendResponse?.msg || '登录失败：无效的响应或服务器错误。');
      }
    } catch (error: unknown) { 
      // catch 块处理 API 调用过程中的意外错误，例如网络问题，
      // 或者 apiService 拦截器未能处理并重新抛出的错误。
      // 后端返回的业务错误 (如 code !== 0) 应该在上面的 try 块中被处理。
      console.error('登录请求遭遇意外错误:', error); 
      let errorMessage = '登录发生意外错误，请稍后重试。'; // 默认的错误消息
      
      // 尝试从 AxiosError 中提取更具体的信息
      // eslint-disable-next-line @typescript-eslint/no-explicit-any -- 允许any类型以检查isAxiosError等属性
      if (typeof error === 'object' && error !== null && 'isAxiosError' in error && (error as any).isAxiosError) {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any -- 允许any类型以访问response和message
        const axiosError = error as any; // 断言为 any 以访问 Axios 特有属性
        if (axiosError.response?.data?.msg) {
          // 如果后端在错误响应中提供了 msg 字段
          errorMessage = axiosError.response.data.msg;
        } else if (axiosError.message) { 
          // 如果没有 response.data.msg，尝试使用 AxiosError 的顶层 message 属性
            errorMessage = axiosError.message;
        }
      } else if (error instanceof Error) { 
        // 处理其他标准的 JavaScript Error 对象
        errorMessage = error.message;
      }
      
      message.error(errorMessage); // 向用户显示最终确定的错误消息
    } finally {
      setLoading(false); // 无论成功或失败，确保在 finally 块中将加载状态设置为 false
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
