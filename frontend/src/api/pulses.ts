import { apiClient } from './apiClient';
import { API_URL } from './config';
import { BalancePoint } from './forecast';

export interface PulseCard {
  id: number;
  name: string;
  created_at: string;
  as_of?: string;
  balance: number;
  history: BalancePoint[];
}

export const listPulses = () =>
  apiClient<PulseCard[]>(`${API_URL}/api/pulses`, { method: 'GET' });

export const getPulse = (pulseId: string) =>
  apiClient<PulseCard>(`${API_URL}/api/pulses/${pulseId}`, { method: 'GET' });

export const createPulse = (name: string, file: File) => {
  const body = new FormData();
  body.append('name', name);
  body.append('file', file);
  return apiClient<PulseCard>(`${API_URL}/api/pulses`, { method: 'POST', body });
};

export const deletePulse = (pulseId: number) =>
  apiClient<void>(`${API_URL}/api/pulses/${pulseId}`, { method: 'DELETE' });
