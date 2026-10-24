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
}

// ==================== НОРМАЛИЗАЦИЯ ПОЛЬЗОВАТЕЛЯ ====================

export function normalizeUser(rawUser: any): UserInfo {
  if (!rawUser) {
    return {
      id: 0,
      email: '',
      company_name: '',
      created_at: '',
      updated_at: '',
      name: '',
      createdAt: '',
    };
  }

  const companyName = rawUser.company_name || rawUser.name || '';
  const createdAt = rawUser.created_at || rawUser.createdAt || new Date().toISOString();
  const updatedAt = rawUser.updated_at || rawUser.updatedAt || createdAt;

  return {
    id: rawUser.id,
    email: rawUser.email,
    company_name: companyName,
    created_at: createdAt,
    updated_at: updatedAt,
    name: companyName,
    createdAt: createdAt,
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



  try {
  const createdUser = await apiClient<UserInfo>(`${API_URL}/register`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });

  return normalizeUser(createdUser);
  } catch (err) {
    const msg = err instanceof Error ? err.message : '';
    if (msg.includes('already exists') || msg.includes('23505') || msg.includes('duplicate')) {
      throw new Error('Эта почта уже зарегистрирована');
    }
    if (msg === 'failed to create user') {
      throw new Error('Не удалось создать пользователя. Попробуйте другую почту.');
    }
    throw err;
  }
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
 * Получение профиля текущего пользователя через /me
 */
export const getMe = async (): Promise<UserInfo> => {
  const res = await apiClient<UserInfo>(`${API_URL}/me`, {
    method: 'GET',
  });
  return normalizeUser(res);
};

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