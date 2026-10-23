import  {apiClient}  from './apiClient';
import {API_URL}  from "./config";

export interface LoginData {
  email: string;
  password: string;
}

export interface RegisterData {
  name: string;
  email: string;
  password: string;
  repeatPassword: string;
}


export interface AuthResponse {
  user: User;
}


export interface User {
  id: number;
  name: string;
  email: string;
  imageUrl?: string | null;
  createdAt: string;
}




const isPasswordValid = (password: string) =>
  /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^A-Za-z0-9]).{8,}$/.test(password);

const isUserNameValid = (username: string) =>
  /^[a-z0-9_-]{3,16}$/.test(username);

const isEmailValid = (email: string) =>
  /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);

// ==================== ЛОГИН ====================

export  const  login = async (data: LoginData) => {
  return apiClient<AuthResponse>(`${API_URL}/api/login`, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// ==================== РЕГИСТРАЦИЯ ====================
export const registration =async  (data:RegisterData) => { //async что бы throw не вылетал до создание промиса
   if (!isEmailValid(data.email)) {
    throw new Error('Неверный формат email');
  }
  if (!isUserNameValid(data.name)) {
    throw new Error('Имя: 3-16 символов, латиница, цифры, _ и -');
  }
  if (!isPasswordValid(data.password)) {
    throw new Error('Пароль: минимум 8 символов, заглавная, строчная, цифра, спецсимвол');
  }
  if (data.password !== data.repeatPassword) {
    throw new Error('Пароли не совпадают');
  }
  return  apiClient<AuthResponse>(`${API_URL}/api/register`,
    { method: 'POST', body: JSON.stringify(data)} );
 
}

// ==================== ПРОВЕРКА СЕССИИ ====================

export const getMe = () =>
  apiClient<AuthResponse>(`${API_URL}/api/me`);;

// ==================== ЛОГАУТ ====================

export const logoutApi = ()=> {

  return apiClient<AuthResponse>(`${API_URL}/api/logout`, {method: 'POST'});
}

// ==================== Смена имени ====================

export const updateName = (name: string) =>
  apiClient<AuthResponse>(`${API_URL}/api/me`, {
    method: 'PATCH',
    body: JSON.stringify({ name }),
  });

  // ==================== Смена пароля ====================

  export const changePassword = (data: { oldPassword: string; newPassword: string }) =>
  apiClient<void>(`${API_URL}/api/me/password`, {
    method: 'POST',
    body: JSON.stringify(data),
  });


//====================Загрузка Картинки профиля =============
export const updateAvatar = (file: File) => {
  const formData = new FormData();
  formData.append('avatar', file);   // 'avatar' — имя поля, которое ждёт сервер
  return apiClient<AuthResponse>(`${API_URL}/api/upload-avatar`, {method: 'POST', body: formData});
}