
import { API_URL } from "./config";

type UnauthorizedHandler = () => void;
type ErrorHandler = (message: string) => void;

const BASE_URL = API_URL;

let onUnauthorized: UnauthorizedHandler | null = null;
let onError: ErrorHandler | null = null;

// регистраторы — вызываются из AuthProvider
export const setUnauthorizedHandler = (fn: UnauthorizedHandler) => {
  onUnauthorized = fn;
};
export const setErrorHandler = (fn: ErrorHandler) => {
  onError = fn;
};

export async function apiClient<T>(
  url: string,
  options: RequestInit = {}
): Promise<T> {
  let res: Response;
  const fullUrl = url.startsWith('http') ? url : `${BASE_URL}${url}`;
  try {
    res = await fetch(fullUrl, {
      credentials: 'include',        // важно: чтобы cookie уходили
      ...options,
      headers: {
        ...(options.body instanceof FormData
          ? {}                        // для FormData Content-Type ставит сам браузер
          : { 'Content-Type': 'application/json' }),
        ...options.headers,
      },
    });
  } catch (e) {
    const msg = 'Сервер недоступен. Проверьте соединение.';
    onError?.(msg);
    throw new Error(msg);
  }

  // 401 — сессия истекла
  if (res.status === 401) {
    onUnauthorized?.();
    throw new Error('Сессия истекла, войдите заново');
  }

  // 500 и прочие ошибки сервера
  if (res.status >= 500) {
    const msg = 'Ошибка сервера. Попробуйте позже.';
    onError?.(msg);
    throw new Error(msg);
  }

  // 4xx (валидация и т.п.) — пробуем достать сообщение из тела
  if (!res.ok) {
    let msg = 'Произошла ошибка';
    try {
      const data = await res.json();
      msg = data.message || data.error || msg;
    } catch { }
    throw new Error(msg);
  }

  // 204 No Content
  const contentType = res.headers.get('content-type');
  if (!contentType || !contentType.includes('application/json')) {
    return undefined as T;
  }
  return res.json();
}