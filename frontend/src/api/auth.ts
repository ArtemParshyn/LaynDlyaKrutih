import { apiClient, clearTokens, setAccessToken, setRefreshToken } from './apiClient';
import { API_URL } from "./config";

// ==================== SWAGGER DTO И ИНТЕРФЕЙСЫ ====================

export interface UserInfo {
  id: number;
  email: string;
  company_name: string;
  created_at: string;
  updated_at: string;
  // Обратная совместимость для UI компонентов:
  name?: string;
  createdAt?: string;
  imageUrl?: string | null;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  user: UserInfo;
}

export interface CreateUserRequest {
  company_name: string;
  email: string;
  password: string;
  confirm_password: string;
}

export interface UpdateUserRequest {
  company_name?: string;
  password?: string;
}

export interface RefreshTokenRequest {
  refresh_token?: string;
}

export interface RefreshTokenResponse {
  access_token: string;
  refresh_token?: string;
}

export interface SuccessResponse {
  message: string;
}

export interface ErrorResponse {
  error: string;
}

// Алиасы для обратной совместимости с существующими компонентами
export type LoginData = LoginRequest;
export type User = UserInfo;

export interface RegisterData {
  company_name?: string;
  name?: string;
  email: string;
  password: string;
  confirm_password?: string;
  repeatPassword?: string;
}

export interface AuthResponse {
  user: UserInfo;
}

// ==================== НОРМАЛИЗАЦИЯ ПОЛЬЗОВАТЕЛЯ ====================

export function normalizeUser(rawUser: any): UserInfo {
  if (!rawUser) return rawUser;

  const companyName = rawUser.company_name || rawUser.name || '';
  const createdAt = rawUser.created_at || rawUser.createdAt || new Date().toISOString();
  const updatedAt = rawUser.updated_at || rawUser.updatedAt || createdAt;
  const localAvatar = rawUser.id ? localStorage.getItem(`user_avatar_${rawUser.id}`) : null;

  return {
    id: rawUser.id,
    email: rawUser.email,
    company_name: companyName,
    created_at: createdAt,
    updated_at: updatedAt,
    name: companyName,
    createdAt: createdAt,
    imageUrl: localAvatar || rawUser.imageUrl || null,
  };
}

// ==================== ВАЛИДАЦИЯ ====================

const isEmailValid = (email: string) =>
  /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);

// ==================== АВТОРИЗАЦИЯ ====================

/**
 * POST /login
 * Проверяет email и пароль, возвращает Access и Refresh токены, а также сохраняет сессию в Redis
 */
export const login = async (data: LoginRequest): Promise<LoginResponse> => {
  const res = await apiClient<LoginResponse>(`${API_URL}/login`, {
    method: 'POST',
    body: JSON.stringify({
      email: data.email.trim(),
      password: data.password,
    }),
  });

  if (res.access_token) {
    setAccessToken(res.access_token);
  }
  if (res.refresh_token) {
    setRefreshToken(res.refresh_token);
  }

  return {
    ...res,
    user: normalizeUser(res.user),
  };
};

// ==================== РЕГИСТРАЦИЯ ====================

/**
 * POST /register
 * Создает нового пользователя и возвращает его данные (UserInfo)
 */
export const registration = async (data: RegisterData): Promise<UserInfo> => {
  const companyName = (data.company_name || data.name || '').trim();
  const email = (data.email || '').trim();
  const password = data.password || '';
  const confirmPassword = data.confirm_password || data.repeatPassword || '';

  if (!isEmailValid(email)) {
    throw new Error('Неверный формат email');
  }
  if (!companyName || companyName.length < 2 || companyName.length > 64) {
    throw new Error('Название компании / имя должно содержать от 2 до 64 символов');
  }
  if (password.length < 6) {
    throw new Error('Пароль должен содержать минимум 6 символов');
  }
  if (password !== confirmPassword) {
    throw new Error('Пароли не совпадают');
  }

  const payload: CreateUserRequest = {
    company_name: companyName,
    email,
    password,
    confirm_password: confirmPassword,
  };

  const createdUser = await apiClient<UserInfo>(`${API_URL}/register`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });

  return normalizeUser(createdUser);
};

// ==================== ПРОФИЛЬ ТЕКУЩЕГО ПОЛЬЗОВАТЕЛЯ ====================

/**
 * GET /me
 * Возвращает информацию об авторизованном пользователе (BearerAuth)
 */
export const getMe = async (): Promise<UserInfo> => {
  const user = await apiClient<UserInfo>(`${API_URL}/me`, {
    method: 'GET',
  });
  return normalizeUser(user);
};

// ==================== ВЫХОД ИЗ СИСТЕМЫ ====================

/**
 * POST /logout
 * Удаляет Refresh сессию из Redis и очищает cookies
 */
export const logoutApi = async (): Promise<SuccessResponse> => {
  try {
    const res = await apiClient<SuccessResponse>(`${API_URL}/logout`, {
      method: 'POST',
    });
    return res;
  } finally {
    clearTokens();
  }
};

// ==================== ОБНОВЛЕНИЕ ТОКЕНА ====================

/**
 * POST /refresh
 * Принимает Refresh токен из Cookie или body, выпускает новый Access токен
 */
export const refreshTokenApi = async (data?: RefreshTokenRequest): Promise<RefreshTokenResponse> => {
  return apiClient<RefreshTokenResponse>(`${API_URL}/refresh`, {
    method: 'POST',
    body: JSON.stringify(data || {}),
  });
};

// ==================== CRUD ПОЛЬЗОВАТЕЛЕЙ ====================

/**
 * GET /users
 * Возвращает список всех пользователей из кэша Redis или базы PostgreSQL (BearerAuth)
 */
export const getAllUsers = async (): Promise<UserInfo[]> => {
  const users = await apiClient<UserInfo[]>(`${API_URL}/users`, {
    method: 'GET',
  });
  return users.map(normalizeUser);
};

/**
 * GET /users/{id}
 * Возвращает данные пользователя по ID (BearerAuth)
 */
export const getUserById = async (id: number): Promise<UserInfo> => {
  const user = await apiClient<UserInfo>(`${API_URL}/users/${id}`, {
    method: 'GET',
  });
  return normalizeUser(user);
};

/**
 * PATCH /users/{id}
 * Обновляет пароль и/или название компании пользователя (BearerAuth)
 */
export const updateUser = async (id: number, data: UpdateUserRequest): Promise<UserInfo> => {
  const user = await apiClient<UserInfo>(`${API_URL}/users/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  });
  return normalizeUser(user);
};

/**
 * DELETE /users/{id}
 * Удаляет пользователя из базы PostgreSQL и сбрасывает кэш в Redis (BearerAuth)
 */
export const deleteUser = async (id: number): Promise<SuccessResponse> => {
  return apiClient<SuccessResponse>(`${API_URL}/users/${id}`, {
    method: 'DELETE',
  });
};

// ==================== ХЕЛПЕРЫ ДЛЯ СОВМЕСТИМОСТИ ====================

/**
 * Обновление названия компании пользователя через PATCH /users/{id}
 */
export const updateName = async (name: string, userId?: number): Promise<UserInfo> => {
  if (!userId) {
    throw new Error('ID пользователя не указан');
  }
  return updateUser(userId, { company_name: name });
};

/**
 * Обновление пароля пользователя через PATCH /users/{id}
 */
export const changePassword = async (
  data: { oldPassword?: string; newPassword: string },
  userId?: number
): Promise<UserInfo> => {
  if (!userId) {
    throw new Error('ID пользователя не указан');
  }
  if (!data.newPassword || data.newPassword.length < 6) {
    throw new Error('Новый пароль должен быть не менее 6 символов');
  }
  return updateUser(userId, { password: data.newPassword });
};

/**
 * Локальное сохранение аватарки профиля (так как на бэкенде нет endpoint загрузки файлов)
 */
export const updateAvatar = async (file: File, userId?: number): Promise<{ user: UserInfo }> => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const base64 = reader.result as string;
      if (userId) {
        localStorage.setItem(`user_avatar_${userId}`, base64);
      }
      resolve({
        user: {
          id: userId || 0,
          email: '',
          company_name: '',
          created_at: '',
          updated_at: '',
          imageUrl: base64,
        },
      });
    };
    reader.onerror = () => reject(new Error('Не удалось прочитать файл изображения'));
    reader.readAsDataURL(file);
  });
};