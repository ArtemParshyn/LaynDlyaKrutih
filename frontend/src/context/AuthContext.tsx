import { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react';
import {
  login as apiLogin,
  registration as apiRegister,
  logoutApi,
  updateUser as apiUpdateUser,
  deleteUser as apiDeleteUser,
  normalizeUser,
  LoginRequest,
  RegisterData,
  UserInfo,
} from '../api/auth';
import {
  setUnauthorizedHandler,
  setErrorHandler,
  getAccessToken,
  decodeAccessToken,
  clearTokens,
} from '../api/apiClient';
import { toast } from 'sonner';

interface AuthContextType {
  user: UserInfo | null;
  isAuthenticated: boolean;
  loading: boolean;
  login: (data: LoginRequest) => Promise<void>;
  register: (data: RegisterData) => Promise<void>;
  logout: () => Promise<void>;
  isModalOpen: boolean;
  openModal: () => void;
  closeModal: () => void;
  changePassword: (data: string | { oldPassword?: string; newPassword: string }) => Promise<void>;
  updateName: (name: string) => Promise<void>;
  deleteAccount: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

const userFromAccessToken = (): UserInfo | null => {
  const token = getAccessToken();
  if (!token) return null;

  const claims = decodeAccessToken(token);
  if (!claims) return null;
  if (claims.exp && claims.exp * 1000 <= Date.now()) return null;

  const id = Number(claims.user_id ?? claims.sub);
  if (!Number.isFinite(id)) return null;

  return normalizeUser({
    id,
    email: claims.email || '',
    company_name: claims.company_name || '',
    created_at: claims.iat ? new Date(claims.iat * 1000).toISOString() : '',
    updated_at: '',
  });
};

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [isModalOpen, setModalOpen] = useState(false);
  const [user, setUser] = useState<UserInfo | null>(null);
  const [loading, setLoading] = useState(true);

  // Обработчик 401 Unauthorized
  const handleUnauthorized = useCallback(() => {
    setUser(null);
    clearTokens();
    toast.error('Сессия истекла. Пожалуйста, войдите снова.');
  }, []);

  useEffect(() => {
    setErrorHandler((msg) => toast.error(msg));
    setUnauthorizedHandler(handleUnauthorized);

    return () => {
      setUnauthorizedHandler(() => {});
      setErrorHandler(() => {});
    };
  }, [handleUnauthorized]);

  // Профиль читается из access JWT (user_id, email, company_name). Запроса /me нет.
  useEffect(() => {
    const current = userFromAccessToken();
    if (!current && getAccessToken()) clearTokens();
    setUser(current);
    setLoading(false);
  }, []);

  const login = async (data: LoginRequest) => {
    await apiLogin(data);
    setUser(userFromAccessToken());
  };

  const register = async (data: RegisterData) => {
    await apiRegister(data);
    // Автоматический логин после успешной регистрации для мгновенного входа
    try {
      await apiLogin({
        email: data.email,
        password: data.password,
      });
      setUser(userFromAccessToken());
    } catch {
      // Если авто-логин не прошел, просим войти вручную
    }
  };

  const logout = async () => {
    try {
      await logoutApi();
    } catch {
      // Игнорируем ошибку выхода на сервере
    } finally {
      clearTokens();
      setUser(null);
    }
  };

  const openModal = () => setModalOpen(true);
  const closeModal = () => setModalOpen(false);

  const changePassword = async (data: string | { oldPassword?: string; newPassword: string }) => {
    if (!user?.id) throw new Error('Пользователь не авторизован');
    const newPassword = typeof data === 'string' ? data : data.newPassword;
    if (!newPassword || newPassword.length < 6) {
      throw new Error('Пароль должен содержать не менее 6 символов');
    }

    const updated = await apiUpdateUser(user.id, { password: newPassword });
    setUser(userFromAccessToken() || updated);
  };

  const updateName = async (name: string) => {
    if (!user?.id) throw new Error('Пользователь не авторизован');
    const trimmed = name.trim();
    if (trimmed.length < 2 || trimmed.length > 64) {
      throw new Error('Название компании должно быть от 2 до 64 символов');
    }

    const updated = await apiUpdateUser(user.id, { company_name: trimmed });
    setUser({ ...(userFromAccessToken() || updated), company_name: trimmed, name: trimmed });
  };

  const deleteAccount = async () => {
    if (!user?.id) throw new Error('Пользователь не авторизован');
    await apiDeleteUser(user.id);
    clearTokens();
    setUser(null);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        loading,
        login,
        register,
        logout,
        isModalOpen,
        openModal,
        closeModal,
        changePassword,
        updateName,
        deleteAccount,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth должен использоваться внутри AuthProvider');
  }
  return context;
};

export default AuthContext;