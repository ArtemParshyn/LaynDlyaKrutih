import { API_URL } from "./config";

type UnauthorizedHandler = () => void;
type ErrorHandler = (message: string) => void;

const BASE_URL = API_URL;

const ACCESS_TOKEN_KEY = 'access_token';
const REFRESH_TOKEN_KEY = 'refresh_token';

export const getAccessToken = (): string | null => {
  return localStorage.getItem(ACCESS_TOKEN_KEY);
};

export const setAccessToken = (token: string | null) => {
  if (token) {
    localStorage.setItem(ACCESS_TOKEN_KEY, token);
  } else {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
  }
};

export const getRefreshToken = (): string | null => {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
};

export const setRefreshToken = (token: string | null) => {
  if (token) {
    localStorage.setItem(REFRESH_TOKEN_KEY, token);
  } else {
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  }
};

export const clearTokens = () => {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
};

export interface AccessTokenClaims {
  user_id?: number;
  email?: string;
  company_name?: string;
  sub?: string;
  exp?: number;
  iat?: number;
}

/** Читает payload JWT без проверки подписи. Подпись проверяет бэкенд. */
export const decodeAccessToken = (token: string): AccessTokenClaims | null => {
  const payload = token.split('.')[1];
  if (!payload) return null;
  try {
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=');
    return JSON.parse(atob(padded)) as AccessTokenClaims;
  } catch {
    return null;
  }
};

let onUnauthorized: UnauthorizedHandler | null = null;
let onError: ErrorHandler | null = null;

// Регистраторы обработчиков (вызываются из AuthProvider)
export const setUnauthorizedHandler = (fn: UnauthorizedHandler) => {
  onUnauthorized = fn;
};

export const setErrorHandler = (fn: ErrorHandler) => {
  onError = fn;
};

interface RequestOptions extends RequestInit {
  _isRetry?: boolean;
}

export async function apiClient<T>(
  url: string,
  options: RequestOptions = {}
): Promise<T> {
  const fullUrl = url.startsWith('http') ? url : `${BASE_URL}${url}`;
  const token = getAccessToken();

  const headers: Record<string, string> = {
    ...(options.body instanceof FormData
      ? {}
      : { 'Content-Type': 'application/json' }),
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...(options.headers as Record<string, string> || {}),
  };

  let res: Response;
  try {
    res = await fetch(fullUrl, {
      credentials: 'include', // важно для cookies и Redis-сессий
      ...options,
      headers,
    });
  } catch (e) {
    const msg = 'Сервер недоступен. Проверьте соединение с бэкендом.';
    onError?.(msg);
    throw new Error(msg);
  }

  // 401 — Unauthorized: пробуем обновить токен через /refresh
  if (res.status === 401) {
    const refreshToken = getRefreshToken();
    if (!options._isRetry && refreshToken) {
      try {
        const refreshRes = await fetch(`${BASE_URL}/refresh`, {
          method: 'POST',
          credentials: 'include',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refreshToken }),
        });

        if (refreshRes.ok) {
          const refreshData = await refreshRes.json();
          if (refreshData.access_token) {
            setAccessToken(refreshData.access_token);
            if (refreshData.refresh_token) {
              setRefreshToken(refreshData.refresh_token);
            }
            // Повторяем запрос с новым access токеном
            return apiClient<T>(url, {
              ...options,
              _isRetry: true,
            });
          }
        }
      } catch {
        // Если refresh завершился ошибкой сети или сервера
      }
    }

    clearTokens();
    onUnauthorized?.();
    throw new Error('Сессия истекла, войдите заново');
  }

  // 500 и прочие ошибки сервера
  if (res.status >= 500) {
    let msg = 'Ошибка сервера. Попробуйте позже.';
    try {
      const errData = await res.json();
      msg = errData.error || errData.message || msg;
    } catch { }
    onError?.(msg);
    throw new Error(msg);
  }

  // 4xx (валидация, конфликт 409 и т.п.)
  if (!res.ok) {
    let msg = 'Произошла ошибка';
    try {
      const data = await res.json();
      msg = data.error || data.message || msg;
    } catch { }
    throw new Error(msg);
  }

  // 204 No Content или не JSON
  const contentType = res.headers.get('content-type');
  if (!contentType || !contentType.includes('application/json')) {
    return undefined as T;
  }

  return res.json();
}