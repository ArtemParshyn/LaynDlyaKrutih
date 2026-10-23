import { createContext, useContext, useState, useEffect, ReactNode, useCallback } from 'react';
import {
  login as apiLogin,
  registration as apiRegister,
  getMe,
  logoutApi,
  updateUser as apiUpdateUser,
  deleteUser as apiDeleteUser,
  updateAvatar as apiUploadAvatar,
  LoginRequest,
  RegisterData,
  UserInfo,
} from '../api/auth';
import {
  setUnauthorizedHandler,
  setErrorHandler,
  getAccessToken,
  getRefreshToken,
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
  updateAvatar: (file: File) => Promise<void>;
  isModalOpen: boolean;
  openModal: () => void;
  closeModal: () => void;
  changePassword: (data: string | { oldPassword?: string; newPassword: string }) => Promise<void>;
  updateName: (name: string) => Promise<void>;
  deleteAccount: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

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

  // Проверка сессии при загрузке приложения
  useEffect(() => {
    const checkAuth = async () => {
      const token = getAccessToken() || getRefreshToken();
      if (!token) {
        setLoading(false);
        return;
      }

      try {
        const currentUser = await getMe();
        setUser(currentUser);
      } catch {
        setUser(null);
        clearTokens();
      } finally {
        setLoading(false);
      }
    };

    checkAuth();
  }, []);

  const login = async (data: LoginRequest) => {
    const result = await apiLogin(data);
    setUser(result.user);
  };

  const register = async (data: RegisterData) => {
    await apiRegister(data);
    // Автоматический логин после успешной регистрации для мгновенного входа
    try {
      const loginResult = await apiLogin({
        email: data.email,
        password: data.password,
      });
      setUser(loginResult.user);
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

  const updateAvatar = async (file: File) => {
    if (!user) return;
    const res = await apiUploadAvatar(file, user.id);
    setUser((prev) => (prev ? { ...prev, imageUrl: res.user.imageUrl } : null));
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
    setUser(updated);
  };

  const updateName = async (name: string) => {
    if (!user?.id) throw new Error('Пользователь не авторизован');
    const trimmed = name.trim();
    if (trimmed.length < 2 || trimmed.length > 64) {
      throw new Error('Название компании должно быть от 2 до 64 символов');
    }

    const updated = await apiUpdateUser(user.id, { company_name: trimmed });
    setUser(updated);
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
        updateAvatar,
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