import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import {
  login as apiLogin,
  registration as apiRegister,
  getMe,
  logoutApi,
  LoginData,
  RegisterData,
  AuthResponse,
  updateAvatar as apiUploadAvatar,
  changePassword as apiChangePassword,
  updateName as apiUpdateName,
} from '../api/auth';
import { setUnauthorizedHandler, setErrorHandler } from '../api/apiClient'
import { toast } from 'sonner';
import { useNavigate } from "react-router-dom";

interface AuthContextType {
  user: AuthResponse['user'] | null;
  isAuthenticated: boolean;
  loading: boolean;   // ← пока проверяем сессию, показываем лоадер
  login: (data: LoginData) => Promise<void>;
  register: (data: RegisterData) => Promise<void>;
  logout: () => Promise<void>;
  updateAvatar: (file: File) => Promise<void>;
  isModalOpen: boolean;
  openModal: () => void;
  closeModal: () => void;
  changePassword: (data: { oldPassword: string; newPassword: string }) => Promise<void>;
  updateName: (name: string) => void;
}

const AuthContext = createContext<AuthContextType | null>(null);


export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [isModalOpen, setModalOpen] = useState(false)
  const [user, setUser] = useState<AuthResponse['user'] | null>(null);
  const [loading, setLoading] = useState(true);   // ← пока не проверили сессию — true

  useEffect(() => {
    setErrorHandler((msg) => toast.error(msg));
    setUnauthorizedHandler(() => {
      setUser(null);
      toast.error('Сессия истекла')
    });
  }, []);



  useEffect(() => {
    const checkAuth = async () => {
      try {
        const result = await getMe();
        setUser(result.user);
      }
      catch {
        setUser(null)
      }
      finally {
        setLoading(false);
      }
    };
    checkAuth();
  }, []);

  const login = async (data: LoginData) => {
    const result = await apiLogin(data);

    setUser(result.user);

  };

  const register = async (data: RegisterData) => {
    const result = await apiRegister(data);
    setUser(result.user);
  };

  useEffect(() => {
    setErrorHandler((msg) => toast.error(msg));
    setUnauthorizedHandler(() => {
      setUser(null);
      toast.error('Сессия истекла');
    });

    return () => {
      setUnauthorizedHandler(() => { });
      setErrorHandler(() => { });
    };
  }, []);

  const logout = async () => {
    try {
      await logoutApi();
    } catch {
    } finally {
      setUser(null);
    }
  };

  const updateAvatar = async (file: File) => {
    const result = await apiUploadAvatar(file);
    setUser(result.user);
  };
  const openModal = () => { setModalOpen(true) }
  const closeModal = () => { setModalOpen(false) }
  const changePassword = async (data: { oldPassword: string; newPassword: string }) => {
    const result = await apiChangePassword(data);

  }

  const updateName = async (name: string) => {
    const result = await apiUpdateName(name);
    setUser(result.user);
  }

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